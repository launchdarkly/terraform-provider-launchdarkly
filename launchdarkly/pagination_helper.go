package launchdarkly

import "fmt"

// offsetPage captures one offset/limit API page.
// totalCount is optional because some API responses omit it.
type offsetPage[T any] struct {
	items      []T
	totalCount *int64
}

func offsetPageFromInt32Ptr[T any](items []T, totalCount *int32) offsetPage[T] {
	var totalCount64 *int64
	if totalCount != nil {
		count := int64(*totalCount)
		totalCount64 = &count
	}
	return offsetPage[T]{
		items:      items,
		totalCount: totalCount64,
	}
}

func offsetPageFromInt32[T any](items []T, totalCount int32) offsetPage[T] {
	count := int64(totalCount)
	return offsetPage[T]{
		items:      items,
		totalCount: &count,
	}
}

// fetchAllOffsetPages collects items from an offset/limit paginated endpoint.
// It supports both totalCount-based responses and count-less responses.
func fetchAllOffsetPages[T any](
	limit int64,
	initialOffset int64,
	fetchPage func(offset, limit int64) (offsetPage[T], error),
) ([]T, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("pagination limit must be greater than zero, got %d", limit)
	}

	allItems := make([]T, 0)
	offset := initialOffset

	for {
		page, err := fetchPage(offset, limit)
		if err != nil {
			return allItems, err
		}

		fetchedCount := int64(len(page.items))
		allItems = append(allItems, page.items...)

		// Safety net: stop if API returns no progress.
		if fetchedCount == 0 {
			break
		}

		if page.totalCount != nil {
			if offset+fetchedCount >= *page.totalCount {
				break
			}
		} else if fetchedCount < limit {
			break
		}

		offset += fetchedCount
	}

	return allItems, nil
}

// fetchAllOffsetPagesWithOptionalInt32Total adapts endpoints that return items + optional *int32 totalCount.
func fetchAllOffsetPagesWithOptionalInt32Total[T any](
	limit int64,
	initialOffset int64,
	fetchPage func(offset, limit int64) (items []T, totalCount *int32, err error),
) ([]T, error) {
	return fetchAllOffsetPages[T](limit, initialOffset, func(offset, limit int64) (offsetPage[T], error) {
		items, totalCount, err := fetchPage(offset, limit)
		if err != nil {
			return offsetPage[T]{}, err
		}
		return offsetPageFromInt32Ptr(items, totalCount), nil
	})
}

// fetchAllOffsetPagesWithInt32Total adapts endpoints that return items + required int32 totalCount.
func fetchAllOffsetPagesWithInt32Total[T any](
	limit int64,
	initialOffset int64,
	fetchPage func(offset, limit int64) (items []T, totalCount int32, err error),
) ([]T, error) {
	return fetchAllOffsetPages[T](limit, initialOffset, func(offset, limit int64) (offsetPage[T], error) {
		items, totalCount, err := fetchPage(offset, limit)
		if err != nil {
			return offsetPage[T]{}, err
		}
		return offsetPageFromInt32(items, totalCount), nil
	})
}
