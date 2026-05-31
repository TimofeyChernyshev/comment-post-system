package graph

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	"github.com/graph-gophers/dataloader/v7"
)

type ctxKey struct{}

func WithLoaders(ctx context.Context, l *Loader) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func Loaders(ctx context.Context) *Loader {
	return ctx.Value(ctxKey{}).(*Loader)
}

type ChildrenKey struct {
	ParentID string
	First    int
	After    *string
}

func (k ChildrenKey) String() string {
	b, _ := json.Marshal(k)
	return string(b)
}

type ChildrenResult struct {
	Items       []*domain.Comment
	HasNextPage bool
}

type Loader struct {
	commentChildren *dataloader.Loader[ChildrenKey, ChildrenResult]
}

func NewChildrenLoader(commentService CommentService, batchCapacity int, waitTime time.Duration) *Loader {
	batchFn := func(ctx context.Context, keys []ChildrenKey) []*dataloader.Result[ChildrenResult] {
		parentIDs := make([]string, len(keys))
		firsts := make([]int, len(keys))
		afters := make([]*string, len(keys))

		for i, k := range keys {
			parentIDs[i] = k.ParentID
			firsts[i] = k.First
			afters[i] = k.After
		}

		res, hasNext, err := commentService.ListChildrenBatch(ctx, parentIDs, firsts, afters)
		results := make([]*dataloader.Result[ChildrenResult], len(keys))

		for i, k := range keys {
			if err != nil {
				results[i] = &dataloader.Result[ChildrenResult]{Error: err}
				continue
			}

			results[i] = &dataloader.Result[ChildrenResult]{
				Data: ChildrenResult{
					Items:       res[k.ParentID],
					HasNextPage: hasNext[k.ParentID],
				},
			}
		}

		return results
	}

	return &Loader{
		commentChildren: dataloader.NewBatchedLoader(
			batchFn,
			dataloader.WithBatchCapacity[ChildrenKey, ChildrenResult](batchCapacity),
			dataloader.WithWait[ChildrenKey, ChildrenResult](waitTime),
		),
	}
}
