package db

import (
	"fmt"
)

// PaginationOptions represents pagination parameters
type PaginationOptions struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	OrderBy  string `json:"order_by"`
	Order    string `json:"order"` // "asc" or "desc"
}

// DefaultPaginationOptions returns default pagination settings
func DefaultPaginationOptions() PaginationOptions {
	return PaginationOptions{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		Order:    "desc",
	}
}

// Validate checks if pagination options are valid
func (p PaginationOptions) Validate() error {
	if p.Page < 1 {
		return fmt.Errorf("page must be >= 1")
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		return fmt.Errorf("page_size must be between 1 and 100")
	}
	if p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("order must be 'asc' or 'desc'")
	}
	return nil
}

// BuildOrderClause builds the ORDER BY clause for SQL queries
func (p PaginationOptions) BuildOrderClause() string {
	if p.OrderBy == "" {
		p.OrderBy = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
	return fmt.Sprintf("ORDER BY %s %s", p.OrderBy, p.Order)
}

// BuildLimitClause builds the LIMIT/OFFSET clause for SQL queries
func (p PaginationOptions) BuildLimitClause() string {
	offset := (p.Page - 1) * p.PageSize
	return fmt.Sprintf("LIMIT %d OFFSET %d", p.PageSize, offset)
}

// PaginatedResponse represents a paginated response
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse[T any](data []T, options PaginationOptions, totalItems int) *PaginatedResponse[T] {
	totalPages := (totalItems + options.PageSize - 1) / options.PageSize
	return &PaginatedResponse[T]{
		Data:       data,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
