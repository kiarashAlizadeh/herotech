package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
)

func parsePagination(c *gin.Context) (dto.PaginationRequest, error) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		return dto.PaginationRequest{}, errors.New("invalid page parameter")
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 {
		return dto.PaginationRequest{}, errors.New("invalid page_size parameter")
	}

	return dto.PaginationRequest{
		Page:     page,
		PageSize: pageSize,
	}, nil
}
