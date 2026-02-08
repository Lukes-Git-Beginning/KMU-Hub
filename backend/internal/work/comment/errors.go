package comment

import "errors"

var (
	ErrNotFound                = errors.New("comment not found")
	ErrContentRequired         = errors.New("comment content is required")
	ErrContentTooLong          = errors.New("comment content must be at most 10000 characters")
	ErrQuotedCommentNotFound   = errors.New("quoted comment not found")
	ErrQuotedCommentWrongTask  = errors.New("quoted comment belongs to a different task")
	ErrCannotEditOthersComment = errors.New("cannot edit another user's comment")
	ErrCannotDeleteOthersComment = errors.New("cannot delete another user's comment")
)
