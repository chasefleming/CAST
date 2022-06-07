package strategies

import (
	"fmt"

	"github.com/brudfyi/flow-voting-tool/main/models"
)

type OneAddressOneVote struct{}

func (s *OneAddressOneVote) TallyVotes(votes []*models.VoteWithBalance, p *models.ProposalResults) (models.ProposalResults, error) {

	//tally votes
	for _, vote := range votes {
		p.Results[vote.Choice]++
	}

	return *p, nil
}

func (s *OneAddressOneVote) GetVotes(votes []*models.VoteWithBalance, proposal *models.Proposal) ([]*models.VoteWithBalance, error) {

	for _, vote := range votes {
		weight, err := s.GetVoteWeightForBalance(vote, proposal)
		if err != nil {
			return nil, err
		}
		vote.Weight = &weight
	}

	return votes, nil
}

func (s *OneAddressOneVote) GetVoteWeightForBalance(vote *models.VoteWithBalance, proposal *models.Proposal) (float64, error) {
	var weight float64
	var ERROR error = fmt.Errorf("no address found")

	if vote.Addr == "" {
		return 0.00, ERROR
	}
	weight = 1.00

	return weight, nil
}
