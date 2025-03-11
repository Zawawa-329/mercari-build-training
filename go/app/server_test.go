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

	"github.com/google/go-cmp/cmp"
	"github.com/golang/mock/gomock" 
)

func TestParseAddItemRequest(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
    	t.Fatalf("failed to get current working directory: %v", err)
	}
	t.Logf("Current working directory: %s", cwd)


	type wants struct {
		req *AddItemRequest
		err bool
	}

	imageBytes, err := os.ReadFile("/home/saway/mercari-build-training/go/images/default.jpg")
	if err != nil {
		t.Fatalf("failed to read image file: %v", err)
	} else {
		t.Log("Image file loaded successfully")
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


// STEP 6-4: uncomment this test
// func TestAddItemE2e(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping e2e test")
// 	}

// 	db, closers, err := setupDB(t)
// 	if err != nil {
// 		t.Fatalf("failed to set up database: %v", err)
// 	}
// 	t.Cleanup(func() {
// 		for _, c := range closers {
// 			c()
// 		}
// 	})

// 	type wants struct {
// 		code int
// 	}
// 	cases := map[string]struct {
// 		args map[string]string
// 		wants
// 	}{
// 		"ok: correctly inserted": {
// 			args: map[string]string{
// 				"name":     "used iPhone 16e",
// 				"category": "phone",
// 			},
// 			wants: wants{
// 				code: http.StatusOK,
// 			},
// 		},
// 		"ng: failed to insert": {
// 			args: map[string]string{
// 				"name":     "",
// 				"category": "phone",
// 			},
// 			wants: wants{
// 				code: http.StatusBadRequest,
// 			},
// 		},
// 	}

// 	for name, tt := range cases {
// 		t.Run(name, func(t *testing.T) {
// 			h := &Handlers{itemRepo: &itemRepository{db: db}}

// 			values := url.Values{}
// 			for k, v := range tt.args {
// 				values.Set(k, v)
// 			}
// 			req := httptest.NewRequest("POST", "/items", strings.NewReader(values.Encode()))
// 			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 			rr := httptest.NewRecorder()
// 			h.AddItem(rr, req)

// 			// check response
// 			if tt.wants.code != rr.Code {
// 				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
// 			}
// 			if tt.wants.code >= 400 {
// 				return
// 			}
// 			for _, v := range tt.args {
// 				if !strings.Contains(rr.Body.String(), v) {
// 					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
// 				}
// 			}

// 			// STEP 6-4: check inserted data
// 		})
// 	}
// }

// func setupDB(t *testing.T) (db *sql.DB, closers []func(), e error) {
// 	t.Helper()

// 	defer func() {
// 		if e != nil {
// 			for _, c := range closers {
// 				c()
// 			}
// 		}
// 	}()

// 	// create a temporary file for e2e testing
// 	f, err := os.CreateTemp(".", "*.sqlite3")
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	closers = append(closers, func() {
// 		f.Close()
// 		os.Remove(f.Name())
// 	})

// 	// set up tables
// 	db, err = sql.Open("sqlite3", f.Name())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	closers = append(closers, func() {
// 		db.Close()
// 	})

// 	// TODO: replace it with real SQL statements.
// 	cmd := `CREATE TABLE IF NOT EXISTS items (
// 		id INTEGER PRIMARY KEY AUTOINCREMENT,
// 		name VARCHAR(255),
// 		category VARCHAR(255)
// 	)`
// 	_, err = db.Exec(cmd)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return db, closers, nil
// }
