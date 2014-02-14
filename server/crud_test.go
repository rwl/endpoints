
package server

import (
	"time"
	"net/http"
	"github.com/rwl/go-endpoints/endpoints"
	"github.com/rwl/go-endpoints/endpoints/google"
)

var db = map[auth.User]map[int64]BlogPost

type BlogPost struct {
	Id int64 `json:"id"`
	Name string `json:"name"`
	Date time.Time `json:"date"`
	Published bool `json:"published"`
	Content string `json:"content"`
}

type BlogService struct {}

func (s *BlogService) Delete(r *http.Request) error {
	return nil
}

func (s *BlogService) Get(r *http.Request) error {
	return nil
}

func (s *BlogService) Insert(r *http.Request) error {
	return nil
}

func (s *BlogService) List(r *http.Request) error {
	return nil
}

func (s *BlogService) Patch(r *http.Request) error {
	return nil
}

func (s *BlogService) Update(r *http.Request) error {
	return nil
}

func ExampleCrud() {
	endpoints.AddAuthProvider(google.NewAuthProvider())
	spi := endpoints.NewServer("")

	BlogService := &BlogService{}
	blogApi, _ := spi.RegisterService(BlogService, "blog",
		"v1", "Blog API", true)

	del := blogApi.MethodByName("Delete").Info()
	del.Name = "blog.delete"
	del.HttpMethod = "DELETE"
	del.Path = "delete"
	del.Desc = "Deletes a blog post."

	get := blogApi.MethodByName("Get").Info()
	get.Name = "blog.get"
	get.HttpMethod = "GET"
	get.Path = "get"
	get.Desc = "Gets a blog post by ID."

	insert := blogApi.MethodByName("Insert").Info()
	insert.Name = "blog.insert"
	insert.HttpMethod = "POST"
	insert.Path = "insert"
	insert.Desc = "Creates a new blog post."

	list := blogApi.MethodByName("List").Info()
	list.Name = "blog.list"
	list.HttpMethod = "GET"
	list.Path = "list"
	list.Desc = "Lists all existing blog posts."

	patch := blogApi.MethodByName("Patch").Info()
	patch.Name = "blog.patch"
	patch.HttpMethod = "PATCH"
	patch.Path = "patch"
	patch.Desc = "Updates and existing post. This method supports patch semantics."

	update := blogApi.MethodByName("Update").Info()
	update.Name = "blog.update"
	update.HttpMethod = "PUT"
	update.Path = "update"
	update.Desc = "Updates and existing post."

}
