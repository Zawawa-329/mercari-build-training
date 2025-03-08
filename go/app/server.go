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
	// ロガーの設定
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// データベースのセットアップ
	db, err := setupDatabase()
	if err != nil {
		slog.Error("failed to setup database", "error", err)
		return 1
	}

	// アイテムリポジトリの作成
	itemRepo := NewItemRepository(db)
	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// ルーティングの設定
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Hello)
	mux.HandleFunc("GET /items", h.GetItems)
	mux.HandleFunc("POST /items", h.AddItem)
	mux.HandleFunc("GET /images/{filename}", h.GetImage)
	mux.HandleFunc("GET /items/{item_id}", h.GetItem)
	mux.HandleFunc("GET /search",h.SearchItems)

	// サーバ起動
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
	Name string `form:"name"`
	Category string `form:"category`
	Image []byte
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the request to add an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, error) {
	req := &AddItemRequest{
		Name: r.FormValue("name"),
		Category: r.FormValue("category"),
	}
	// STEP 4-4: add an image field

	// validate the request
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	if req.Category == "" {
		return nil, errors.New("category is required")
	}
	file, _, err := r.FormFile("image")
	if err != nil {
    	return nil, errors.New("failed to read image file")
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
    	return nil, errors.New("failed to read image data")
	}
	req.Image = imageData

	return req, nil
}


func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := parseAddItemRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 画像を保存してファイル名を取得
	fileName, err := s.storeImage(req.Image)
	if err != nil {
		slog.Error("failed to store image: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	item := &Item{
		Name:          req.Name,
		Category:      req.Category,
		ImageFileName: fileName,
	}

	// アイテムをデータベースに挿入
	err = s.itemRepo.Insert(ctx, item)
	if err != nil {
		slog.Error("failed to store item: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// アイテム一覧を取得
	items, err := s.itemRepo.LoadItems()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"items": items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}


func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.itemRepo.LoadItems()
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
	fileName := fmt.Sprintf("%x.jpg", hash) // ハッシュを 16 進数文字列に変換

    filePath := filepath.Join(s.imgDirPath, fileName)

    // ハッシュ化された画像ファイル名を作成
    //hash.Write(image)  // imageをハッシュ化する
    //hashValue := hash.Sum256(nil)
    //hashedFileName := fmt.Sprintf("%s.jpg", hex.EncodeToString(hashValue))

    // 保存先のパスを作成
    //filePath = filepath.Join(s.imgDirPath, hashedFileName)

    // ファイルを保存
    outFile, err := os.Create(filePath)
    if err != nil {
        return "", fmt.Errorf("failed to create image file: %w", err)
    }
    defer outFile.Close()

    // 画像データを書き込む
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

		// デフォルト画像を返す
		imgPath = filepath.Join(s.imgDirPath, "default.jpg")
	}

	slog.Info("returned image", "path", imgPath)
	http.ServeFile(w, r, imgPath)
}


func (r *itemRepository) LoadItems() ([]*Item, error) {
	query := `SELECT id, name, category, image_name FROM items`
	rows, err := r.db.Query(query)
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
			// 画像が見つからなかった場合にログを出力
			slog.Info("Image not found: ", "path", imgPath)
			return filepath.Join(s.imgDirPath, "default.jpg"), nil  // Default image path
		}
		return "", errImageNotFound
	}

	return imgPath, nil
}
// GetItem は指定された ID のアイテムを取得するエンドポイント
type GetItemResponse struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	ImageName  string `json:"image_name"`
}

// GetItem は指定された ID のアイテムを取得するエンドポイント
func (s *Handlers) GetItem(w http.ResponseWriter, r *http.Request) {
	// item_id を URL から取得
	idStr := r.PathValue("item_id") // "/items/" の後ろが item_id
	id, err := strconv.Atoi(idStr) // 文字列を整数に変換
	if err != nil {
		http.Error(w, "invalid item_id", http.StatusBadRequest)
		return
	}

	items, err := s.itemRepo.LoadItems()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// item_id 番目のアイテムを探す
	if id < 1 || id > len(items) {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	// ID 番目のアイテムを取得
	item := items[id-1]

	// アイテムの詳細を JSON で返す
	resp := GetItemResponse{
		ID:        item.ID,
		Name:      item.Name,
		Category:  item.Category,
		ImageName: item.ImageFileName, // 画像ファイル名を返す
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ItemRepositoryにSearchItemsByNameメソッドを追加
func (r *itemRepository) SearchItemsByName(keyword string) ([]*Item, error) {
	// SQLクエリでLIKEを使用して部分一致検索
	query := `SELECT id, name, category, image_name FROM items WHERE name LIKE ?`
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

// SearchItemsはGET /searchエンドポイントのハンドラー
func (s *Handlers) SearchItems(w http.ResponseWriter, r *http.Request) {
	// クエリパラメータからキーワードを取得
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		http.Error(w, "キーワードのクエリパラメータが必要です", http.StatusBadRequest)
		return
	}

	// アイテムをキーワードで検索
	items, err := s.itemRepo.SearchItemsByName(keyword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// レスポンス構造体を作成
	resp := map[string]interface{}{
		"items": items,
	}

	// JSONとしてレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
