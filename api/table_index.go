package api

import (
	"context"
	"encoding/json"
)

type Index struct {
	ID int `jsonapi:"primary,index"`

	// Standard fields
	CreatorId int
	CreatedTs int64 `json:"createdTs"`
	UpdaterId int
	UpdatedTs int64 `json:"updatedTs"`

	// Related fields
	DatabaseId int
	TableId    int

	// Domain specific fields
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Position   int    `json:"position"`
	Type       string `json:"type"`
	Unique     bool   `json:"unique"`
	Visible    bool   `json:"visible"`
	Comment    string `json:"comment"`
}

type IndexCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	DatabaseId int
	TableId    int

	// Domain specific fields
	Name       string
	Expression string
	Position   int
	Type       string
	Unique     bool
	Visible    bool
	Comment    string
}

type IndexFind struct {
	ID *int

	// Related fields
	DatabaseId *int
	TableId    *int

	// Domain specific fields
	Name       *string
	Expression *string
}

func (find *IndexFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type IndexPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int
}

type IndexService interface {
	CreateIndex(ctx context.Context, create *IndexCreate) (*Index, error)
	FindIndexList(ctx context.Context, find *IndexFind) ([]*Index, error)
	FindIndex(ctx context.Context, find *IndexFind) (*Index, error)
	PatchIndex(ctx context.Context, patch *IndexPatch) (*Index, error)
}
