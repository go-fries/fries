package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type args struct {
	page    int
	prePage int
	total   int
}

type want struct {
	page     int
	prePage  int
	total    int
	lastPage int
	offset   int
	hasMore  bool
	prevPage int
	nextPage int
}

func TestPaginator(t *testing.T) {
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "basic",
			args: args{page: 1, prePage: 10, total: 100},
			want: want{page: 1, prePage: 10, total: 100, lastPage: 10, offset: 0, hasMore: true, prevPage: 1, nextPage: 2},
		},
		{
			name: "the last page",
			args: args{page: 10, prePage: 10, total: 100},
			want: want{page: 10, prePage: 10, total: 100, lastPage: 10, offset: 90, hasMore: false, prevPage: 9, nextPage: 10},
		},
		{
			name: "the last page with different prePage",
			args: args{page: 10, prePage: 11, total: 100},
			want: want{page: 10, prePage: 11, total: 100, lastPage: 10, offset: 99, hasMore: false, prevPage: 9, nextPage: 10},
		},
		{
			name: "zero page",
			args: args{page: 0, prePage: 10, total: 100},
			want: want{page: 1, prePage: 10, total: 100, lastPage: 10, offset: 0, hasMore: true, prevPage: 1, nextPage: 2},
		},
		{
			name: "zero prePage",
			args: args{page: 1, prePage: 0, total: 100},
			want: want{page: 1, prePage: 10, total: 100, lastPage: 10, offset: 0, hasMore: true, prevPage: 1, nextPage: 2},
		},
		{
			name: "zero total",
			args: args{page: 1, prePage: 10, total: 0},
			want: want{page: 1, prePage: 10, total: 0, lastPage: 1, offset: 0, hasMore: false, prevPage: 1, nextPage: 1},
		},
		{
			name: "negative page",
			args: args{page: -1, prePage: 10, total: 100},
			want: want{page: 1, prePage: 10, total: 100, lastPage: 10, offset: 0, hasMore: true, prevPage: 1, nextPage: 2},
		},
		{
			name: "page * prePage > total",
			args: args{page: 10, prePage: 11, total: 100},
			want: want{page: 10, prePage: 11, total: 100, lastPage: 10, offset: 99, hasMore: false, prevPage: 9, nextPage: 10},
		},
		{
			name: "page > lastPage",
			args: args{page: 10, prePage: 10, total: 10},
			want: want{page: 1, prePage: 10, total: 10, lastPage: 1, offset: 0, hasMore: false, prevPage: 1, nextPage: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPaginator(tt.args.page, tt.args.prePage, tt.args.total)

			assert.Equal(t, tt.want.page, p.GetPage())
			assert.Equal(t, tt.want.prePage, p.GetPerPage())
			assert.Equal(t, tt.want.total, p.GetTotal())
			assert.Equal(t, tt.want.lastPage, p.GetLastPage())
			assert.Equal(t, tt.want.offset, p.GetOffset())
			assert.Equal(t, tt.want.hasMore, p.HasMore())
			assert.Equal(t, tt.want.prevPage, p.GetPrevPage())
			assert.Equal(t, tt.want.nextPage, p.GetNextPage())
		})
	}
}

func TestPaginator_ToJSON(t *testing.T) {
	p := NewPaginator(1, 10, 100)
	bytes, err := p.ToJSON()
	assert.NoError(t, err)
	assert.JSONEq(t, `{"page":1,"pre_page":10,"total":100,"last_page":10,"offset":0,"has_more":true,"prev_page":1,"next_page":2}`, string(bytes)) //nolint:lll
}
