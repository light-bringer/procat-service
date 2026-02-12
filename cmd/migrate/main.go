package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	projectID  = flag.String("project", getEnvOrDefault("SPANNER_PROJECT_ID", "test-project"), "GCP project ID")
	instanceID = flag.String("instance", getEnvOrDefault("SPANNER_INSTANCE_ID", "dev-instance"), "Spanner instance ID")
	databaseID = flag.String("database", getEnvOrDefault("SPANNER_DATABASE_ID", "product-catalog-db"), "Spanner database ID")
	migrateDir = flag.String("migrations", "migrations", "Directory containing migration SQL files")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	// Check if using emulator
	emulatorHost := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorHost != "" {
		log.Printf("Using Spanner emulator at %s", emulatorHost)
	}

	if err := run(ctx); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migrations completed successfully!")
}

func run(ctx context.Context) error {
	// Ensure instance exists
	if err := ensureInstance(ctx); err != nil {
		return fmt.Errorf("failed to ensure instance: %w", err)
	}

	if err := ensureDatabase(ctx); err != nil {
		return fmt.Errorf("failed to ensure database: %w", err)
	}

	// Apply migrations
	if err := applyMigrations(ctx); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func ensureInstance(ctx context.Context) error {
	log.Printf("Ensuring instance %s exists...", *instanceID)

	instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create instance admin client: %w", err)
	}
	defer instanceAdmin.Close()

	instanceName := fmt.Sprintf("projects/%s/instances/%s", *projectID, *instanceID)

	// Check if instance exists
	_, err = instanceAdmin.GetInstance(ctx, &instancepb.GetInstanceRequest{
		Name: instanceName,
	})

	if err == nil {
		log.Println("Instance already exists")
		return nil
	}

	// Create instance if it doesn't exist
	if status.Code(err) == codes.NotFound {
		log.Println("Creating instance...")
		op, err := instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
			Parent:     fmt.Sprintf("projects/%s", *projectID),
			InstanceId: *instanceID,
			Instance: &instancepb.Instance{
				Config:      fmt.Sprintf("projects/%s/instanceConfigs/emulator-config", *projectID),
				DisplayName: "Development Instance",
				NodeCount:   1,
			},
		})
		if err != nil {
			// Ignore if already exists
			if status.Code(err) != codes.AlreadyExists {
				return fmt.Errorf("failed to create instance: %w", err)
			}
			log.Println("Instance already exists")
			return nil
		}

		// Don't wait too long on emulator
		if _, err := op.Wait(ctx); err != nil {
			// Emulator might complete immediately, ignore certain errors
			if status.Code(err) != codes.AlreadyExists {
				log.Printf("Warning during instance creation: %v", err)
			}
		}

		log.Println("Instance created successfully")
		return nil
	}

	log.Printf("Warning: unexpected error checking instance: %v", err)
	return nil
}

func ensureDatabase(ctx context.Context) error {
	log.Printf("Ensuring database %s exists...", *databaseID)

	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", *projectID, *instanceID, *databaseID)

	// Check if database exists
	_, err = adminClient.GetDatabase(ctx, &databasepb.GetDatabaseRequest{
		Name: dbPath,
	})

	if err == nil {
		log.Println("Database already exists")
		return nil
	}

	// Create database if it doesn't exist
	if status.Code(err) == codes.NotFound {
		log.Println("Creating database...")
		op, err := adminClient.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
			Parent:          fmt.Sprintf("projects/%s/instances/%s", *projectID, *instanceID),
			CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", *databaseID),
		})
		if err != nil {
			// Ignore if database already exists
			if status.Code(err) != codes.AlreadyExists {
				return fmt.Errorf("failed to create database: %w", err)
			}
			log.Println("Database already exists")
			return nil
		}

		if _, err := op.Wait(ctx); err != nil {
			return fmt.Errorf("failed to wait for database creation: %w", err)
		}

		log.Println("Database created successfully")
		return nil
	}

	// For other errors on emulator, just proceed - the DB might exist
	if os.Getenv("SPANNER_EMULATOR_HOST") != "" {
		log.Printf("Proceeding with database (emulator mode): %v", err)
		return nil
	}

	return fmt.Errorf("failed to check database: %w", err)
}

func applyMigrations(ctx context.Context) error {
	log.Printf("Applying migrations from %s...", *migrateDir)

	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	// Get list of migration files
	files, err := filepath.Glob(filepath.Join(*migrateDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}

	if len(files) == 0 {
		log.Println("No migration files found")
		return nil
	}

	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", *projectID, *instanceID, *databaseID)

	for _, file := range files {
		migrationName := filepath.Base(file)
		log.Printf("Applying %s...", migrationName)

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Split into individual DDL statements
		statements := splitDDLStatements(string(content))

		// Apply DDL statements
		op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   dbPath,
			Statements: statements,
		})
		if err != nil {
			return fmt.Errorf("failed to start DDL update for %s: %w", migrationName, err)
		}

		if err := op.Wait(ctx); err != nil {
			return fmt.Errorf("failed to apply DDL for %s: %w", migrationName, err)
		}

		log.Printf("Successfully applied %s", migrationName)
	}

	return nil
}

func splitDDLStatements(content string) []string {
	// Remove comments and empty lines
	lines := strings.Split(content, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		cleaned = append(cleaned, line)
	}

	content = strings.Join(cleaned, "\n")

	// Split by semicolon
	statements := strings.Split(content, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	return result
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
