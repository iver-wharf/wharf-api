package modelconv

import (
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/request"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"gopkg.in/typ.v3/pkg/sets"
)

// DBBranchListToResponse converts a list of branches and an optional default
// branch to a response list of branches.
func DBBranchListToResponse(dbAllBranches []database.Branch, dbDefaultBranch *database.Branch) response.BranchList {
	resBranchList := response.BranchList{
		Branches: DBBranchesToResponses(dbAllBranches),
	}
	if dbDefaultBranch != nil {
		resDefaultBranch := DBBranchToResponse(*dbDefaultBranch)
		resBranchList.DefaultBranch = &resDefaultBranch
	}
	return resBranchList
}

// DBBranchListToPaginatedResponse converts a list of branches and an optional
// default branch to a response paginated list of branches.
func DBBranchListToPaginatedResponse(dbBranches []database.Branch, allBranchesCount int64, dbDefaultBranch *database.Branch) response.PaginatedBranches {
	resPaginatedBranches := response.PaginatedBranches{
		List:       DBBranchesToResponses(dbBranches),
		TotalCount: allBranchesCount,
	}
	if dbDefaultBranch != nil {
		resDefaultBranch := DBBranchToResponse(*dbDefaultBranch)
		resPaginatedBranches.DefaultBranch = &resDefaultBranch
	}
	return resPaginatedBranches
}

// DBBranchesToResponses converts a slice of database branches to a slice of
// response branches.
func DBBranchesToResponses(dbBranches []database.Branch) []response.Branch {
	resBranches := make([]response.Branch, len(dbBranches))
	for i, dbBranch := range dbBranches {
		resBranches[i] = DBBranchToResponse(dbBranch)
	}
	return resBranches
}

// DBBranchToResponse converts a database branch to a response branch.
func DBBranchToResponse(dbBranch database.Branch) response.Branch {
	return response.Branch{
		TimeMetadata: DBTimeMetadataToResponse(dbBranch.TimeMetadata),
		BranchID:     dbBranch.BranchID,
		ProjectID:    dbBranch.ProjectID,
		Name:         dbBranch.Name,
		Default:      dbBranch.Default,
		TokenID:      dbBranch.TokenID,
	}
}

// ReqBranchUpdatesToSetOfNames converts a slice of request branch updates to a
// set of branch names.
func ReqBranchUpdatesToSetOfNames(reqBranches []request.BranchUpdate) sets.Set[string] {
	namesSet := make(sets.Set[string])
	for _, reqBranchUpdate := range reqBranches {
		namesSet.Add(reqBranchUpdate.Name)
	}
	return namesSet
}

// DBBranchesToSetOfNames converts a slice of database branches to a set of
// branch names.
func DBBranchesToSetOfNames(dbBranches []database.Branch) sets.Set[string] {
	namesSet := make(sets.Set[string])
	for _, dbOldBranch := range dbBranches {
		namesSet.Add(dbOldBranch.Name)
	}
	return namesSet
}
