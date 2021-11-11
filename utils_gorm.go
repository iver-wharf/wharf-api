package main

import "gorm.io/gorm"

func findDBPaginatedSliceAndTotalCount(dbQuery *gorm.DB, limit, offset int, slice interface{}, totalCount *int64) error {
	err := dbQuery.Scopes(optionalLimitOffsetScope(limit, offset)).Find(slice).Error
	if err != nil {
		return err
	}

	return dbQuery.Count(totalCount).Error
}
