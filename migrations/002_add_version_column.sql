-- Migration: Add version column for optimistic locking
--
-- This migration adds a version column to the products table to prevent lost updates
-- from concurrent modifications. The version is incremented on every update.

ALTER TABLE products ADD COLUMN version INT64 NOT NULL DEFAULT (0);
