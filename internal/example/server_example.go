package example

import (
	"fmt"
	"net/http"
)

// 模拟用户类型
type User struct {
	ID   string
	Name string
}

// 模拟用户鉴权函数
func authenticate(r *http.Request) *User {
	// TODO: 增加鉴权逻辑
	// 这里仅返回 nil 作为示例（表示未通过鉴权）
	return nil
}

// 处理请求的中间件
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := authenticate(r)
		if user == nil {
			fmt.Println("Forbidden!")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "Forbidden!")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// 示例处理函数
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!")
}

func StartServer() {
	// 创建路由并应用鉴权中间件
	mux := http.NewServeMux()
	mux.HandleFunc("/", helloHandler)
	protectedHandler := authMiddleware(mux)

	// 启动服务器
	port := "8080"
	fmt.Printf("Server is starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, protectedHandler); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
