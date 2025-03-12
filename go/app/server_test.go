package app

import (
	"net/http"
	"net/http/httptest"
	//"net/url"
	"strings"
	"testing"
	"os"
	"bytes"
	"mime/multipart"
	"errors"
	"database/sql"
	//"io"
	"fmt"
	

	"github.com/google/go-cmp/cmp"
	"github.com/golang/mock/gomock" 
	_ "github.com/mattn/go-sqlite3"
)

func TestParseAddItemRequest(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
    	t.Fatalf("failed to get current working directory: %v", err)
	}

	type wants struct {
		req *AddItemRequest
		err bool
	}

	imageBytes, err := os.ReadFile("/home/saway/mercari-build-training/go/images/default.jpg")
	if err != nil {
		t.Fatalf("failed to read image file: %v", err)
	} 
	// STEP 6-1: define test cases
	cases := map[string]struct {
		args 		map[string]string
		imageData	[]byte
		wants
	}{
		"ok: valid request": {
			args: map[string]string{
				"name":     "jacket", // fill here
				"category": "fashion", // fill here
				"image":	"default.jpg",
			},
			imageData: imageBytes, 
			wants: wants{
				req: &AddItemRequest{
					Name: 		"jacket", // fill here
					Category:	"fashion", // fill here
					Image:		imageBytes,
				},
				err: false,
			},
		},
		"ng: empty request": {
			args:		map[string]string{},
			imageData:	nil,
			wants: wants{
				req: nil,
				err: true,
			},
		},

		"ng: empty name": {
			args:		map[string]string{
				"category": "fashion",
				"image":    "default.jpg",
			},
			imageData:	imageBytes,
			wants: wants{
				req: nil,
				err: true,
			},
		},

		"ng: empty category": {
			args:		map[string]string{
				"name":		"jacket",
				"image":	"default.jpg",
			},
			imageData:	imageBytes,
			wants: wants{
				req: nil,
				err: true,
			},
		},

		"ng: empty image": {
			args:		map[string]string{
				"name":     "jacket",
				"category": "fashion",
			},
			imageData:	nil,
			wants: wants{
				req: nil,
				err: true,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// prepare request body
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			for key, value := range tt.args {
				writer.WriteField(key, value)
			}

			if tt.imageData != nil {
				part, err := writer.CreateFormFile("image", "default.jpg")
				if err != nil {
					t.Fatalf("failed to create form file: %v", err)
				}
				part.Write(tt.imageData)
			}

			writer.Close()

			// prepare HTTP request
			req, err := http.NewRequest("POST", "http://localhost:9000/items", &b)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// execute test target
			got, err := parseAddItemRequest(req)

			// confirm the result
			if err != nil {
				if !tt.err {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if diff := cmp.Diff(tt.wants.req, got); diff != "" {
				t.Errorf("unexpected request (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelloHandler(t *testing.T) {
	t.Parallel()

	// Please comment out for STEP 6-2
	// predefine what we want
	type wants struct {
		code int               // desired HTTP status code
		body map[string]string // desired body
	}

	want := wants{
	 	code: http.StatusOK,
	 	body: map[string]string{"message": "Hello, world!"},
	}

	// set up test
	req := httptest.NewRequest("GET", "/hello", nil)
	res := httptest.NewRecorder()

	h := &Handlers{}
	h.Hello(res, req)

	// STEP 6-2: confirm the status code
	if res.Code != want.code {
		t.Errorf("expected status code %d, got %d", want.code, res.Code)
	}

	// STEP 6-2: confirm response body
	for _, v := range want.body {
		if !strings.Contains(res.Body.String(), v) {
			t.Errorf("response body does not contain %s, got: %s", v, res.Body.String())
		}
	}
}

func TestAddItem(t *testing.T) {
    t.Parallel()

	imageBytes, err := os.ReadFile("/home/saway/mercari-build-training/go/images/default.jpg")
	if err != nil {
    	t.Fatalf("failed to read image file: %v", err)
	}
    type wants struct {
        code int
    }
    cases := map[string]struct {
        args       map[string]string
        imageData  []byte
        injector   func(m *MockItemRepository)
        wants
    }{
        "ok: correctly inserted": {
            args: map[string]string{
                "name":     "used iPhone 16e",
                "category": "phone",
                "image":    "default.jpg",
            },
            imageData: imageBytes,
            injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
				// succeeded to insert
				expectedImageFileName := "a1ebfdb99c936f09c57dee11ab7fb3a27dcf3b1dd5ef107bd650345c01ee8f11.jpg"
				expectedItem := &Item{
					Name:          "used iPhone 16e",
					Category:      "phone",
					ImageFileName: expectedImageFileName,
				}
			
				m.EXPECT().
					Insert(gomock.Any(), expectedItem).
					Return(nil).Times(1)
			},
			wants: wants{
                code: http.StatusOK,
            },
        },
        "ng: failed to insert": {
            args: map[string]string{
                "name":     "used iPhone 16e",
                "category": "phone",
                "image":    "default.jpg",
            },
            imageData: imageBytes,
            injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
				// failed to insert
				expectedImageFileName := "a1ebfdb99c936f09c57dee11ab7fb3a27dcf3b1dd5ef107bd650345c01ee8f11.jpg"
				expectedItem := &Item{
					Name:          "used iPhone 16e",
					Category:      "phone",
					ImageFileName: expectedImageFileName,  // 画像ファイル名を使用
				}
			
				m.EXPECT().
					Insert(gomock.Any(), expectedItem).   // アイテム挿入
					Return(errors.New("insert failed")).Times(1)  // アイテム挿入が正常に行われる
			},
            wants: wants{
                code: http.StatusInternalServerError,
            },
        },
    }

    for name, tt := range cases {
        t.Run(name, func(t *testing.T) {
            t.Parallel()

			tmpDir, err := os.MkdirTemp("", "test-images-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
				t.Fatalf("temp directory does not exist: %v", err)
			}

            ctrl := gomock.NewController(t)
			defer ctrl.Finish()

            mockIR := NewMockItemRepository(ctrl)
            tt.injector(mockIR)
            h := &Handlers{imgDirPath: tmpDir,itemRepo: mockIR}

			reqbody := &bytes.Buffer{}
            writer := multipart.NewWriter(reqbody)
            for k, v := range tt.args {
                writer.WriteField(k, v)
            }
			
			if tt.imageData != nil {
				part, err := writer.CreateFormFile("image", "test.jpg")
				if err != nil {
					t.Fatalf("failed to create form file: %v", err)
				}
				_, err = part.Write(tt.imageData)
				if err != nil {
					t.Fatalf("failed to write image data: %v", err)
				}
			}
			err = writer.Close()
			if err != nil {
				t.Fatalf("failed to close writer: %v", err)
			}

            req := httptest.NewRequest("POST", "/items", reqbody)
            req.Header.Set("Content-Type", writer.FormDataContentType())

            res := httptest.NewRecorder()

			h.AddItem(res, req)

			if res.Code != tt.wants.code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, res.Code)
			}
        })
    }
}
func TestAddItemE2e(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test")
    }

    db, closers, err := setupDB(t)
    if err != nil {
        t.Fatalf("failed to set up database: %v", err)
    }
    t.Cleanup(func() {
        for _, c := range closers {
            c()
        }
    })

    // 画像読み込み処理
    imageBytes, err := os.ReadFile("/home/saway/mercari-build-training/go/images/default.jpg")
    if err != nil {
        t.Fatalf("failed to read image file: %v", err)
    }

    imgDirPath := "./images"
    if err := createImageDir(); err != nil {
        t.Fatalf("failed to create image directory: %v", err)
    }

    // カテゴリIDを取得またはカテゴリを挿入
    var categoryId int64
    query := `SELECT id FROM categories WHERE name = ?`
    err = db.QueryRow(query, "phone").Scan(&categoryId)

    if err != nil {
        if err == sql.ErrNoRows {
            // カテゴリがない場合、カテゴリを挿入
            insertCategoryQuery := `INSERT INTO categories (name) VALUES (?)`
            res, err := db.Exec(insertCategoryQuery, "phone")
            if err != nil {
                t.Fatalf("failed to insert category: %v", err)
            }
            categoryId, err = res.LastInsertId()
            if err != nil {
                t.Fatalf("failed to get category id after insert: %v", err)
            }
        } else {
            t.Fatalf("failed to get category id: %v", err)
        }
    }

    // テストケースの定義
    type wants struct {
        code int
    }
    cases := map[string]struct {
        args map[string]string
        imageData []byte
        wants
    }{
        "ok: correctly inserted": {
            args: map[string]string{
                "name":     "used iPhone 16e",
                "category": "phone", // ここでカテゴリ名を送信
            },
            imageData: imageBytes,
            wants: wants{
                code: http.StatusOK,
            },
        },
        "ng: failed to insert": {
            args: map[string]string{
                "name":     "",
                "category": "phone",
            },
            imageData: imageBytes,
            wants: wants{
                code: http.StatusBadRequest,
            },
        },
    }

    for name, tt := range cases {
        t.Run(name, func(t *testing.T) {
            h := &Handlers{itemRepo: &itemRepository{db: db}, imgDirPath: imgDirPath}

            body := &bytes.Buffer{}
            writer := multipart.NewWriter(body)

            // テキストフィールドの追加
            err := writer.WriteField("name", tt.args["name"])
            if err != nil {
                t.Fatalf("failed to write name field: %v", err)
            }
            err = writer.WriteField("category", tt.args["category"]) // カテゴリ名をそのまま送信
            if err != nil {
                t.Fatalf("failed to write category field: %v", err)
            }

            // 画像フィールドの追加
            filePart, err := writer.CreateFormFile("image", "default.jpg")
            if err != nil {
                t.Fatalf("failed to create form file: %v", err)
            }
            _, err = filePart.Write(tt.imageData)
            if err != nil {
                t.Fatalf("failed to write image data: %v", err)
            }

            // マルチパートライターのクローズ
            err = writer.Close()
            if err != nil {
                t.Fatalf("failed to close multipart writer: %v", err)
            }

            // リクエストを作成
            req := httptest.NewRequest("POST", "/items", body)
            req.Header.Set("Content-Type", writer.FormDataContentType())

            // マルチパートフォームのパース
            err = req.ParseMultipartForm(10 * 1024 * 1024) // 10MB制限
            if err != nil {
                t.Fatalf("failed to parse multipart form data: %v", err)
            }

			req.Form.Set("category", fmt.Sprintf("%d", categoryId)) // category_idを設定
			req.Form.Set("name", tt.args["name"])

            rr := httptest.NewRecorder()

            // ハンドラーにリクエストを渡す
            h.AddItem(rr, req)

            // レスポンスコードの確認
            if tt.wants.code != rr.Code {
                t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
            }
            if tt.wants.code >= 400 {
                return
            }

            // アイテムが挿入されたか確認
            var item Item
            query := `SELECT items.id, items.name, items.category_id, items.image_name FROM items WHERE LOWER(items.name) LIKE LOWER(?)`
            row := db.QueryRow(query, tt.args["name"])
            err = row.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFileName)
            if err != nil {
                t.Fatalf("failed to query inserted item: %v", err)
            }

            // カテゴリ名の確認
            var categoryName string
            categoryQuery := `SELECT name FROM categories WHERE id = ?`
            err = db.QueryRow(categoryQuery, item.Category).Scan(&categoryName)
            if err != nil {
                t.Fatalf("failed to get category name: %v", err)
            }

            if categoryName != tt.args["category"] {
                t.Errorf("expected category name %s, got %s", tt.args["category"], categoryName)
            }

            // アイテムの確認
            if item.Name != tt.args["name"] {
                t.Errorf("expected name %s, got %s", tt.args["name"], item.Name)
            }

            if item.ImageFileName != "default.jpg" {
                t.Errorf("expected image_name %s, got %s", "default.jpg", item.ImageFileName)
            }
        })
    }
}


func setupDB(t *testing.T) (db *sql.DB, closers []func(), e error) {
	t.Helper()

 	defer func() {
 		if e != nil {
 			for _, c := range closers {
 				c()
 			}
 		}
 	}()

 	// create a temporary file for e2e testing
 	f, err := os.CreateTemp(".", "*.sqlite3")
 	if err != nil {
 		return nil, nil, err
 	}
 	closers = append(closers, func() {
 		f.Close()
 		os.Remove(f.Name())
 	})

 	// set up tables
 	db, err = sql.Open("sqlite3", f.Name())
 	if err != nil {
 		return nil, nil, err
 	}
 	closers = append(closers, func() {
 		db.Close()
 	})

 	// TODO: replace it with real SQL statements.
 	cmd :=  `
	CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS items (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	name TEXT NOT NULL,
    	category_id INTEGER,
    	image_name TEXT,
    	FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
	);`
 	_, err = db.Exec(cmd)
 	if err != nil {
 		return nil, nil, err
 	}

 	return db, closers, nil
}