package app

import (
	"context"
	"database/sql"
	"errors"
	"io/ioutil"
	"fmt"
	"os"
	// STEP 5-1: uncomment this line
	_ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	ImageFileName string `json:"image_name"`
}

func setupDatabase() (*sql.DB, error) {
	// Open SQLite database file
	db, err := sql.Open("sqlite3", "db/mercari.sqlite3")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	// Read the SQL script from items.sql and execute it
	sqlFile, err := ioutil.ReadFile("db/items.sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read the SQL file: %w", err)
	}

	// Execute the SQL script
	_, err = db.Exec(string(sqlFile))
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL script: %w", err)
	}

	return db, nil
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
    LoadItems(ctx context.Context) ([]*Item, error)
	SearchItemsByName(keyword string) ([]*Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
    db *sql.DB
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() (ItemRepository, error) {
	db, err := setupDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to create item repository: %w", err)
	}
	return &itemRepository{db: db}, nil
}

func (r *itemRepository) Insert(ctx context.Context, item *Item) error {
	tx, err := r.db.BeginTx(ctx, nil) // Start transaction
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // Rollback in case of error

	// Get category_id
	var categoryID int
	err = tx.QueryRowContext(ctx, "SELECT id FROM categories WHERE id = ?", item.Category).Scan(&categoryID)
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}
	

	// Insert new data into items table
	query := `INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)`
	res, err := tx.ExecContext(ctx, query, item.Name, categoryID, item.ImageFileName)
	if err != nil {
		return fmt.Errorf("failed to insert item: %w", err)
	}

	// Get the item's ID
	lastID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get item ID: %w", err)
	}
	item.ID = int(lastID)

	// Get the category name
	var categoryName string
	err = tx.QueryRowContext(ctx, "SELECT name FROM categories WHERE id = ?", categoryID).Scan(&categoryName)
	if err != nil {
		return fmt.Errorf("failed to get category name: %w", err)
	}

	// Set the item’s category name
	item.Category = categoryName

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	filePath := "images/" + fileName
    err := os.WriteFile(filePath, image, 0644)
    if err != nil {
        return fmt.Errorf("failed to store image: %w", err)
    }
    return nil
}
