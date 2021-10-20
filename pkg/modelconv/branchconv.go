package modelconv

import (
	"github.com/iver-wharf/wharf-api/internal/set"
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/request"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
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
		BranchID:  dbBranch.BranchID,
		ProjectID: dbBranch.ProjectID,
		Name:      dbBranch.Name,
		Default:   dbBranch.Default,
		TokenID:   dbBranch.TokenID,
	}
}

func ReqBranchUpdatesToSetOfNames(reqBranches []request.BranchUpdate) set.String {
	namesSet := set.String{}
	for _, reqBranchUpdate := range reqBranches {
		namesSet.Set(reqBranchUpdate.Name)
	}
	return namesSet
}

func DBBranchesToSetOfNames(dbBranches []database.Branch) set.String {
	namesSet := set.String{}
	for _, dbOldBranch := range dbBranches {
		namesSet.Set(dbOldBranch.Name)
	}
	return namesSet
}
