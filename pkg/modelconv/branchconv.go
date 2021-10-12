package modelconv

import (
	"github.com/iver-wharf/wharf-api/pkg/model/database"
	"github.com/iver-wharf/wharf-api/pkg/model/response"
)

func DBBranchListToResponse(dbDefaultBranch *database.Branch, dbAllBranches []database.Branch) response.BranchList {
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
