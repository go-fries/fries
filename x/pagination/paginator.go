package pagination

import "encoding/json"

const (
	defaultPerPage = 10
	defaultPage    = 1
)

type Paginator struct {
	page    int
	prePage int
	total   int

	lastPage int
	offset   int
	hasMore  bool
	prevPage int
	nextPage int
}

func NewPaginator(page, prePage, total int) *Paginator {
	p := &Paginator{
		page:    page,
		prePage: prePage,
		total:   total,
	}

	p.init()

	return p
}

func (p *Paginator) init() {
	p.setPrePage()
	p.setLastPage()
	p.setPage()

	p.offset = (p.page - 1) * p.prePage
	p.hasMore = p.page*p.prePage < p.total

	p.prevPage = p.page
	if p.page > 1 {
		p.prevPage = p.page - 1
	}

	p.nextPage = p.page
	if p.HasMore() {
		p.nextPage = p.page + 1
	}
}

func (p *Paginator) setPrePage() {
	if p.prePage < 1 {
		p.prePage = defaultPerPage
	}
}

func (p *Paginator) setLastPage() {
	p.lastPage = (p.total + p.prePage - 1) / p.prePage
	if p.lastPage < 1 {
		p.lastPage = 1
	}
}

func (p *Paginator) setPage() {
	if p.page < 1 {
		p.page = defaultPage
	}

	if p.page > p.lastPage {
		p.page = p.lastPage
	}
}

func (p *Paginator) GetPage() int {
	return p.page
}

func (p *Paginator) GetPerPage() int {
	return p.prePage
}

func (p *Paginator) GetTotal() int {
	return p.total
}

func (p *Paginator) GetLastPage() int {
	return p.lastPage
}

func (p *Paginator) GetOffset() int {
	return p.offset
}

func (p *Paginator) HasMore() bool {
	return p.hasMore
}

func (p *Paginator) GetPrevPage() int {
	return p.prevPage
}

func (p *Paginator) GetNextPage() int {
	return p.nextPage
}

func (p *Paginator) ToMap() map[string]any {
	return map[string]any{
		"page":      p.page,
		"pre_page":  p.prePage,
		"total":     p.total,
		"last_page": p.lastPage,
		"offset":    p.offset,
		"has_more":  p.hasMore,
		"prev_page": p.prevPage,
		"next_page": p.nextPage,
	}
}

func (p *Paginator) ToJSON() ([]byte, error) {
	return json.Marshal(p.ToMap())
}
