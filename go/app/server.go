package app

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"io"
	"path/filepath"
	"strings"
	"strconv" 
	"context"
)

type Server struct {
	// Port is the port number to listen on.
	Port string
	// ImageDirPath is the path to the directory storing images.
	ImageDirPath string
}

type Items struct {
    Items []*Item `json:"items"`
}

// Run is a method to start the server.
// This method returns 0 if the server started successfully, and 1 otherwise.
func (s Server) Run() int {
	// set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// STEP 5-1: set up the database connection
	
	itemRepo, err := NewItemRepository()
	if err != nil {
		slog.Error("failed to create item repository", "error", err)
		return 1
	}

	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Hello)
	mux.HandleFunc("GET /items", h.GetItems)
	mux.HandleFunc("POST /items", h.AddItem)
	mux.HandleFunc("GET /images/{filename}", h.GetImage)
	mux.HandleFunc("GET /items/{item_id}", h.GetItem)
	mux.HandleFunc("GET /search",h.SearchItems)

	// start the server
	err = http.ListenAndServe(":"+s.Port, simpleCORSMiddleware(simpleLoggerMiddleware(mux), frontURL, []string{"GET", "HEAD", "POST", "OPTIONS"}))
	if err != nil {
		slog.Error("failed to start server: ", "error", err)
		return 1
	}

	return 0
}
type Handlers struct {
	// imgDirPath is the path to the directory storing images.
	imgDirPath string
	itemRepo   ItemRepository
}

type HelloResponse struct {
	Message string `json:"message"`
}

// Hello is a handler to return a Hello, world! message for GET / .
func (s *Handlers) Hello(w http.ResponseWriter, r *http.Request) {
	resp := HelloResponse{Message: "Hello, world!"}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type AddItemRequest struct {
	Name			string `form:"name"`
	Category	 	string `json:"category"`
	Image 			[]byte
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the incoming request for adding an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, error) {
    err := r.ParseMultipartForm(10 << 20) // 10MB までのファイルを処理
	if err != nil {
    	return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}
	req := &AddItemRequest{
    	Name:     r.Form.Get("name"), // ここを修正
    	Category: r.Form.Get("category"),
	}

    // Read the image file (Note: this should happen in the AddItem handler, not here)
    imageFile, _, err := r.FormFile("image")
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve image file: %w", err)
    }
    defer imageFile.Close()

    imageData, err := io.ReadAll(imageFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read image data: %w", err)
    }
    req.Image = imageData

    // Input validation
    if req.Name == "" {
        return nil, errors.New("name is required")
    }
    if req.Category == "" {
        return nil, errors.New("category is required")
    }

    return req, nil
}

// AddItem handles the POST request to add a new item
func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    req, err := parseAddItemRequest(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Save the image
    imageFileName := "default.jpg"
    imagePath := fmt.Sprintf("%s/%s", s.imgDirPath, imageFileName)

    err = os.WriteFile(imagePath, req.Image, 0644)
    if err != nil {
        http.Error(w, "Failed to save image", http.StatusInternalServerError)
        return
    }

    // Create the item
    item := &Item{
        Name:          req.Name,
        Category:      req.Category,
        ImageFileName: imageFileName,
    }

    // Insert into the database
    err = s.itemRepo.Insert(ctx, item)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Prepare the response
    resp := map[string]interface{}{
        "item": item,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}


func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	items, err := s.itemRepo.LoadItems(ctx)
	if err != nil {
		slog.Error("failed to get items from DB", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"items": items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}


func createImageDir() error {
    dir := "images"
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        err := os.Mkdir(dir, os.ModePerm)
        if err != nil {
            return fmt.Errorf("failed to create images directory: %w", err)
        }
    }
    return nil
}
func ensureImageDirExists(dir string) error {
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        err := os.MkdirAll(dir, os.ModePerm)
        if err != nil {
            return fmt.Errorf("failed to create images directory: %w", err)
        }
    }
    return nil
}

// storeImage stores an image and returns the file path and an error if any.
// this method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
// storeImage stores an image and returns the file path and an error if any.
func (s *Handlers) storeImage(image []byte) (string, error) {
	if err := ensureImageDirExists(s.imgDirPath); err != nil {
        return "", err
    }
    hash := sha256.Sum256(image)
	fileName := fmt.Sprintf("%x.jpg", hash)
    filePath := filepath.Join(s.imgDirPath, fileName)
    // Create hashed image file name
    //hash.Write(image)  // Hash the image
    //hashValue := hash.Sum256(nil)
    //hashedFileName := fmt.Sprintf("%s.jpg", hex.EncodeToString(hashValue))
    // Create the save path
    //filePath = filepath.Join(s.imgDirPath, hashedFileName)
    // Save the file
    outFile, err := os.Create(filePath)
    if err != nil {
        return "", fmt.Errorf("failed to create image file: %w", err)
    }
    defer outFile.Close()
	// Write the image data
    _, err = outFile.Write(image)
    if err != nil {
        return "", fmt.Errorf("failed to save image: %w", err)
    }
	slog.Info("image saved to", "path", filePath)
    return fileName, nil
}
type GetImageRequest struct {
	FileName string // path value
}
// parseGetImageRequest parses and validates the request to get an image.
func parseGetImageRequest(r *http.Request) (*GetImageRequest, error) {
	req := &GetImageRequest{
		FileName: r.PathValue("filename"), // from path parameter
	}
	// validate the request
	if req.FileName == "" {
		return nil, errors.New("filename is required")
	}
	return req, nil
}
// GetImage is a handler to return an image for GET /images/{filename} .
// If the specified image is not found, it returns the default image.
func (s *Handlers) GetImage(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetImageRequest(r)
	if err != nil {
		slog.Warn("failed to parse get image request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	imgPath, err := s.buildImagePath(req.FileName)
	if err != nil {
		if !errors.Is(err, errImageNotFound) {
			slog.Warn("failed to build image path: ", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.Debug("image not found", "filename", req.FileName)
		// return the default image
		imgPath = filepath.Join(s.imgDirPath, "default.jpg")
	}
	slog.Info("returned image", "path", imgPath)
	http.ServeFile(w, r, imgPath)
}


func (r *itemRepository) LoadItems(ctx context.Context) ([]*Item, error) {
	query := `
		SELECT items.id, items.name, categories.name, items.image_name 
        FROM items 
        JOIN categories ON items.category_id = categories.id
		`
	rows, err := r.db.QueryContext(ctx,query)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve items: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFileName); err != nil {
			return nil, fmt.Errorf("Failed to scan item: %w", err)
		}
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error occurred while loading items: %w", err)
	}

	return items, nil
}




// buildImagePath builds the image path and validates it.
func (s *Handlers) buildImagePath(imageFileName string) (string, error) {
	imgPath := filepath.Join(s.imgDirPath, filepath.Clean(imageFileName))
	// to prevent directory traversal attacks
	rel, err := filepath.Rel(s.imgDirPath, imgPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid image path: %s", imgPath)
	}
	// validate the image suffix
	if !strings.HasSuffix(imgPath, ".jpg") && !strings.HasSuffix(imgPath, ".jpeg") {
		return "", fmt.Errorf("image path does not end with .jpg or .jpeg: %s", imgPath)
	}
	_, err = os.Stat(imgPath)
	if err != nil {
		if os.IsNotExist(err) {
			// log when the image is not found
			slog.Info("Image not found: ", "path", imgPath)
			return filepath.Join(s.imgDirPath, "default.jpg"), nil  // Default image path
		}
		return "", errImageNotFound
	}
	return imgPath, nil
}

func (s *Handlers) GetItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Get item_id from URL
    idStr := r.PathValue("item_id")
    id, err := strconv.Atoi(idStr) 
    if err != nil {
        http.Error(w, "invalid item_id", http.StatusBadRequest)
        return
    }
	slog.Info("Received item_id:", "item_id", id)

   // Get item and category name
    items, err := s.itemRepo.LoadItems(ctx)
    if err != nil {
        if err.Error() == "item not found" {
            http.Error(w, "item not found", http.StatusNotFound)
        } else {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
        return
    }

    
    if len(items) > 0 {
		item := items[id-1] 
	
		resp := Item{
			ID:             item.ID,
			Name:           item.Name,
			Category:       item.Category,
			ImageFileName:  item.ImageFileName,
		}
	
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "Item not found", http.StatusNotFound)
	}
}

// ItemRepository adds the SearchItemsByName method
func (r *itemRepository) SearchItemsByName(keyword string) ([]*Item, error) {
	// SQL query using LIKE for partial match search
	query := ` SELECT items.id, items.name, categories.name, items.image_name 
        FROM items 
        JOIN categories ON items.category_id = categories.id
        WHERE LOWER(items.name) LIKE LOWER(?)`
	likeKeyword := "%" + strings.ToLower(keyword) + "%"

	rows, err := r.db.Query(query, likeKeyword)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve items: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFileName); err != nil {
			return nil, fmt.Errorf("Failed to scan item: %w", err)
		}
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error occurred while loading items: %w", err)
	}

	return items, nil
}

// SearchItems is the handler for the GET /search endpoint
func (s *Handlers) SearchItems(w http.ResponseWriter, r *http.Request) {
	//  Get keyword from query parameter
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		http.Error(w, "Keyword query parameter is required", http.StatusBadRequest)
		return
	}

	// Search items by keyword
	items, err := s.itemRepo.SearchItemsByName(keyword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create response structure
	resp := map[string]interface{}{
		"items": items,
	}

	// Return the response as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}