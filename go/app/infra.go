package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	Name string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
	ImageFileName    string `db:"image" json:"image_name"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
    LoadItems() (*Items, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() ItemRepository {
	return &itemRepository{fileName: "items.json"}
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
    // 現在のアイテムリストを読み込む
    items, err := i.LoadItems()
    if err != nil {
        return fmt.Errorf("failed to load items: %w", err)
    }
	// 新しいアイテムを追加
	items.Items = append(items.Items, item)

    // items.jsonを開いて新しいアイテムリストを書き込む
    file, err := os.OpenFile(i.fileName, os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        return fmt.Errorf("failed to create items.json: %w", err)
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    err = encoder.Encode(items)
    if err != nil {
        return fmt.Errorf("failed to encode items.json: %w", err)
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
