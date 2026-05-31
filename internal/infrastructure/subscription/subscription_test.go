package subscription

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
)

const (
	testBuffer = 10
)

type SubscriptionManagerTestSuite struct {
	suite.Suite
	manager *SubscriptionManager
}

func (s *SubscriptionManagerTestSuite) SetupTest() {
	s.manager = NewSubscriptionManager(testBuffer)
}

func (s *SubscriptionManagerTestSuite) TearDownTest() {
	s.manager.Close()
}

func TestSubscriptionManagerSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionManagerTestSuite))
}

func createTestComment(postID, id, content string) *domain.Comment {
	return &domain.Comment{
		ID:      id,
		PostID:  postID,
		Content: content,
	}
}

func (s *SubscriptionManagerTestSuite) TestSubscribe_Success() {
	ch, unsub, err := s.manager.SubscribeComments("post-1")
	s.Require().NoError(err)
	s.NotNil(ch)
	s.NotNil(unsub)

	comment := createTestComment("post-1", "c1", "hello")
	go s.manager.PublishComment(comment)

	select {
	case received := <-ch:
		s.Equal(comment, received)
	case <-time.After(time.Second):
		s.Fail("timeout waiting for comment")
	}
}

func (s *SubscriptionManagerTestSuite) TestSubscribe_AfterCloseReturnsError() {
	s.manager.Close()
	_, _, err := s.manager.SubscribeComments("post-1")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "closed")
}

func (s *SubscriptionManagerTestSuite) TestPublish_ReceivedBySubscriber() {
	ch, unsub, err := s.manager.SubscribeComments("post-1")
	require.NoError(s.T(), err)
	defer unsub()

	comment := &domain.Comment{
		ID:      "c1",
		PostID:  "post-1",
		Content: "hello",
	}
	s.manager.PublishComment(comment)

	var received *domain.Comment
	assert.Eventually(s.T(), func() bool {
		select {
		case c := <-ch:
			received = c
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, 5*time.Millisecond)

	assert.Equal(s.T(), comment.ID, received.ID)
}

func (s *SubscriptionManagerTestSuite) TestPublish_OnlySamePostSubscribersGetMessage() {
	ch1, unsub1, _ := s.manager.SubscribeComments("post-1")
	defer unsub1()
	ch2, unsub2, _ := s.manager.SubscribeComments("post-2")
	defer unsub2()

	comment := &domain.Comment{ID: "c1", PostID: "post-1"}
	s.manager.PublishComment(comment)

	assert.Eventually(s.T(), func() bool {
		select {
		case <-ch1:
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, 5*time.Millisecond)

	select {
	case <-ch2:
		s.T().Fatal("unexpected message on ch2")
	default:
	}
}

func (s *SubscriptionManagerTestSuite) TestPublish_BufferOverflow_DropsMessage() {
	manager := NewSubscriptionManager(1)
	ch, unsub, err := manager.SubscribeComments("post-1")
	require.NoError(s.T(), err)
	defer unsub()

	first := &domain.Comment{ID: "first", PostID: "post-1"}
	second := &domain.Comment{ID: "second", PostID: "post-1"}

	manager.PublishComment(first)
	manager.PublishComment(second)

	var got *domain.Comment
	assert.Eventually(s.T(), func() bool {
		select {
		case c := <-ch:
			got = c
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, 5*time.Millisecond)
	assert.Equal(s.T(), "first", got.ID)

	select {
	case <-ch:
		s.T().Fatal("second message should have been dropped")
	default:
	}
}

func (s *SubscriptionManagerTestSuite) TestUnsubscribe_StopsReceiving() {
	ch, unsub, _ := s.manager.SubscribeComments("post-1")
	unsub()

	comment := &domain.Comment{ID: "c1", PostID: "post-1"}
	s.manager.PublishComment(comment)

	select {
	case <-ch:
		s.T().Fatal("received message after unsubscribe")
	default:
	}
}

func (s *SubscriptionManagerTestSuite) TestClose_ClosesAllChannels() {
	ch1, _, _ := s.manager.SubscribeComments("post-1")
	ch2, _, _ := s.manager.SubscribeComments("post-2")

	s.manager.Close()

	_, ok1 := <-ch1
	assert.False(s.T(), ok1)
	_, ok2 := <-ch2
	assert.False(s.T(), ok2)
}

func (s *SubscriptionManagerTestSuite) TestClose_Idempotent() {
	s.manager.Close()
	assert.NotPanics(s.T(), func() {
		s.manager.Close()
	})
}

func (s *SubscriptionManagerTestSuite) TestPublish_AfterClose_NoDelivery() {
	ch, unsub, _ := s.manager.SubscribeComments("post-1")
	defer unsub()
	s.manager.Close()

	comment := &domain.Comment{ID: "c1", PostID: "post-1"}
	s.manager.PublishComment(comment)

	select {
	case c, ok := <-ch:
		s.False(ok)
		s.Nil(c)
	default:
		s.T().Fatal("expected channel to be closed and immediately readable")
	}
}

func (s *SubscriptionManagerTestSuite) TestPublish_ConcurrentSubscribers() {
	chs := make([]<-chan *domain.Comment, 3)
	for i := 0; i < 3; i++ {
		ch, unsub, err := s.manager.SubscribeComments("post-1")
		require.NoError(s.T(), err)
		defer unsub()
		chs[i] = ch
	}

	comment := &domain.Comment{ID: "shared", PostID: "post-1"}
	s.manager.PublishComment(comment)

	for _, ch := range chs {
		assert.Eventually(s.T(), func() bool {
			select {
			case c := <-ch:
				return c.ID == "shared"
			default:
				return false
			}
		}, 100*time.Millisecond, 5*time.Millisecond)
	}
}
