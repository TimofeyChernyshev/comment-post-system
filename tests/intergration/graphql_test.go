package intergration

import (
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/suite"

	"github.com/TimofeyChernyshev/comment-post-system/internal/application"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/graphql/graph"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/storage/memory"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/subscription"
)

type GraphQLTestSuite struct {
	suite.Suite
	srv     *handler.Server
	client  *client.Client
	postSvc *application.PostService
	cmntSvc *application.CommentService
}

var (
	maxTitleLenght   = 200
	maxContentLength = 10000
	maxPostPageSize  = 100

	maxCommentLength   = 2000
	maxCommentPageSize = 100
)

func (s *GraphQLTestSuite) SetupTest() {
	postRepo := memory.NewPostRepository()
	cmntRepo := memory.NewCommentRepository()
	txMgr := memory.NewTransactionManager()
	subMgr := subscription.NewSubscriptionManager(10)

	s.postSvc = application.NewPostService(txMgr, postRepo, maxTitleLenght, maxContentLength, maxPostPageSize)
	s.cmntSvc = application.NewCommentService(txMgr, postRepo, cmntRepo, maxCommentLength, maxCommentPageSize)

	resolver := graph.NewResolver(s.postSvc, s.cmntSvc, subMgr)
	s.srv = handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	s.client = client.New(s.srv)
}

func TestGraphQLSuite(t *testing.T) {
	suite.Run(t, new(GraphQLTestSuite))
}

func (s *GraphQLTestSuite) TestCreatePostAndQuery() {
	var resp struct {
		CreatePost struct {
			ID    string
			Title string
		}
	}
	err := s.client.Post(`mutation {
        createPost(params: {title:"Hello", content:"World", authorID:"u1"}) {
            id
            title
        }
    }`, &resp)
	s.Require().NoError(err)
	s.NotEmpty(resp.CreatePost.ID)
	s.Equal("Hello", resp.CreatePost.Title)
}

func (s *GraphQLTestSuite) TestListPosts() {
	s.client.Post(`mutation { createPost(params: {title:"P1", content:"c", authorID:"u1"}) { id } }`, nil)
	s.client.Post(`mutation { createPost(params: {title:"P2", content:"c", authorID:"u1"}) { id } }`, nil)

	var resp struct {
		Posts struct {
			Edges []struct {
				Node struct {
					Title string
				}
			}
		}
	}
	err := s.client.Post(`{ posts(first:2) { edges { node { title } } } }`, &resp)
	s.Require().NoError(err)
	s.Len(resp.Posts.Edges, 2)
}

func (s *GraphQLTestSuite) TestCommentsWithDataLoader() {
	var createPost struct{ CreatePost struct{ ID string } }
	s.client.Post(`mutation { createPost(params: {title:"P", content:"c", authorID:"u1"}) { id } }`, &createPost)
	postID := createPost.CreatePost.ID

	s.client.Post(fmt.Sprintf(`mutation {
        c1: createComment(params: {postID:"%s", authorID:"u1", content:"root1"}) { id }
        c2: createComment(params: {postID:"%s", authorID:"u1", content:"root2"}) { id }
    }`, postID, postID), nil)

	var q struct {
		Post struct {
			Comments struct {
				Edges []struct {
					Node struct {
						Content string
					}
				}
			}
		}
	}
	err := s.client.Post(fmt.Sprintf(`{
        post(id:"%s") {
            comments(first:10) {
                edges { node { content } }
            }
        }
    }`, postID), &q)
	s.Require().NoError(err)
	s.Len(q.Post.Comments.Edges, 2)
}
