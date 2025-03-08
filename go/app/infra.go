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
	// SQLiteデータベースファイルを開く
	db, err := sql.Open("sqlite3", "db/marcari.sqlite3")
	if err != nil {
		return nil, fmt.Errorf("データベースの接続に失敗しました: %w", err)
	}

	// items.sql のSQLスクリプトを読み込んで実行
	sqlFile, err := ioutil.ReadFile("db/items.sql")
	if err != nil {
		return nil, fmt.Errorf("SQLファイルの読み込みに失敗しました: %w", err)
	}

	// SQLスクリプトを実行
	_, err = db.Exec(string(sqlFile))
	if err != nil {
		return nil, fmt.Errorf("SQLスクリプトの実行に失敗しました: %w", err)
	}

	return db, nil
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
    LoadItems() ([]*Item, error)
	SearchItemsByName(keyword string) ([]*Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
    db *sql.DB
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository(db *sql.DB) ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) Insert(ctx context.Context, item *Item) error {
	query := `INSERT INTO items (name, category, image_name) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, item.Name, item.Category, item.ImageFileName)
	if err != nil {
		return fmt.Errorf("アイテムの挿入に失敗しました: %w", err)
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