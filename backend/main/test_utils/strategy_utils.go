package test_utils

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"github.com/brudfyi/flow-voting-tool/main/models"
	"github.com/brudfyi/flow-voting-tool/main/shared"
)

type VoteWithBalance struct {
	models.Vote

	Primary_account_balance uint64 `json:"primaryAccountBalance"`
	Staking_balance         uint64 `json:"stakingBalance"`
	Block_height            uint64 `json:"blockHeight"`

	NFTs []models.NFT
}

func (otu *OverflowTestUtils) TallyResultsForTokenWeightedDefault(proposalId int, votes *[]VoteWithBalance) *models.ProposalResults {
	r := models.ProposalResults{Proposal_id: proposalId}

	r.Results_float = map[string]float64{}
	r.Results_float["a"] = 0
	r.Results_float["b"] = 0

	for _, v := range *votes {
		r.Results_float[v.Choice] += float64(v.Primary_account_balance) * math.Pow(10, -8)
	}

	return &r
}

func (otu *OverflowTestUtils) TallyResultsForStakedTokenWeightedDefault(proposalId int, votes *[]VoteWithBalance) *models.ProposalResults {
	r := models.ProposalResults{Proposal_id: proposalId}

	r.Results_float = map[string]float64{}
	r.Results_float["a"] = 0
	r.Results_float["b"] = 0

	for _, v := range *votes {
		r.Results_float[v.Choice] += float64(v.Staking_balance) * math.Pow(10, -8)
	}

	return &r
}

func (otu *OverflowTestUtils) TallyResultsForOneAddressOneVote(proposalId int, votes *[]VoteWithBalance) *models.ProposalResults {
	r := models.ProposalResults{Proposal_id: proposalId}

	r.Results = map[string]int{}
	r.Results["a"] = 0
	r.Results["b"] = 0

	for _, v := range *votes {
		r.Results[v.Choice]++
	}

	return &r
}

func (otu *OverflowTestUtils) TallyResultsForBalanceOfNfts(proposalId int, votes *[]VoteWithBalance) *models.ProposalResults {
	r := models.ProposalResults{Proposal_id: proposalId}

	r.Results_float = map[string]float64{}
	r.Results_float["a"] = 0
	r.Results_float["b"] = 0

	for _, v := range *votes {
		nfts := len(v.NFTs)
		r.Results_float[v.Choice] += float64(nfts) * math.Pow(10, -8)
	}
	return &r
}

func (otu *OverflowTestUtils) GenerateListOfVotes(proposalId int, count int) *[]VoteWithBalance {
	votes := make([]VoteWithBalance, count)
	choices := []string{"a", "b"}
	for i := 0; i < count; i++ {
		addr := "0x" + strconv.Itoa(i)
		randomNumber := rand.Intn(2)
		choice := choices[randomNumber]
		v := models.Vote{
			Proposal_id: proposalId, Addr: addr, Choice: choice,
		}

		// Balance is 1 FLOW * index
		balance := 100000000 * (i + 1)

		vote := VoteWithBalance{
			Vote:                    v,
			Primary_account_balance: uint64(balance),
			Staking_balance:         uint64(balance * 5), // Make this different so staked/reg strats dont have same results
			Block_height:            uint64(0),
		}

		votes[i] = vote
	}

	return &votes
}

func (otu *OverflowTestUtils) GenerateListOfVotesWithNFTs(proposalId int, count int, contract *shared.Contract) (*[]VoteWithBalance, error) {
	var votes []VoteWithBalance
	choices := []string{"a", "b"}
	for i := 0; i < count; i++ {
		accountName := "user" + strconv.Itoa(i+1)
		otu.SetupAccountForNFTs(accountName)
		otu.MintNFT("user1", accountName)

		addr := otu.ResolveUser(i + 1)
		randomNumber := rand.Intn(2)
		choice := choices[randomNumber]
		v := models.Vote{
			Proposal_id: proposalId, Addr: addr, Choice: choice,
		}

		nftIds, err := otu.Adapter.GetNFTIds(addr, contract)
		if err != nil {
			return nil, err
		}

		vote := otu.CreateNFTVote(v, nftIds, contract)
		votes = append(votes, vote)
	}

	return &votes, nil
}

func (otu *OverflowTestUtils) CreateNFTVote(v models.Vote, ids []interface{}, contract *shared.Contract) VoteWithBalance {
	nfts := []models.NFT{}

	for _, id := range ids {
		idUint, err := strconv.ParseUint(id.(string), 10, 64)
		if err != nil {
			fmt.Println(err)
			return VoteWithBalance{}
		}

		NFT := models.NFT{
			ID:            idUint,
			Contract_addr: *contract.Addr,
		}
		nfts = append(nfts, NFT)
	}

	vote := VoteWithBalance{
		Vote: v,
		NFTs: nfts,
	}

	return vote
}

func (otu *OverflowTestUtils) CreateNFTCollection(account string) {
	otu.O.TransactionFromFile("create_collection").
		SignProposeAndPayAs(account).
		RunPrintEventsFull()
}

func (otu *OverflowTestUtils) MintNFT(signer string, recipient string) {
	otu.O.TransactionFromFile("mint_nft").
		SignProposeAndPayAsService().
		Args(otu.O.Arguments().
			Account(recipient).
			String("name").
			String("description").
			String("thumbnail")).
		RunPrintEventsFull()
}

func (otu *OverflowTestUtils) SetupAccountForNFTs(account string) {
	otu.O.TransactionFromFile("setup_account").
		SignProposeAndPayAs(account).
		RunPrintEventsFull()
}
