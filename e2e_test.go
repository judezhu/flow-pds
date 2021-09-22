package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
)

func TestE2E(t *testing.T) {
	// t.Skip("skipping for now as this requires a flow emulator")

	cfg := getTestCfg()
	a, cleanup := getTestApp(cfg, true)
	defer func() {
		cleanup()
	}()

	jsonPath := "./flow.json"
	var flowJSON []string = []string{jsonPath}
	g := gwtf.NewGoWithTheFlow(flowJSON, "emulator", false, 3)

	issuer := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "issuer")))
	// pds := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "pds")))
	// owner := common.FlowAddress(flow.HexToAddress(util.GetAccountAddr(g, "owner")))

	// Issuer create PackIssuer resource to store DistCap

	createPackIssuer := "./cadence-transactions/pds/create_new_pack_issuer.cdc"
	createPackIssuerCode := util.ParseCadenceTemplate(createPackIssuer)
	_, err := g.TransactionFromFile(createPackIssuer, createPackIssuerCode).
		SignProposeAndPayAs("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Issuer create PackNFT collection resource to store minted PackNFT

	createPackNFTCollection := "./cadence-transactions/packNFT/create_new_packNFT_collection.cdc"
	createPackNFTCollectionCode := util.ParseCadenceTemplate(createPackNFTCollection)
	_, err = g.TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Owner create PackNFT collection resource to store PackNFT after purchase

	_, err = g.TransactionFromFile(createPackNFTCollection, createPackNFTCollectionCode).
		SignProposeAndPayAs("owner").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// -- Mint example NFTs as issuer --

	mintExampleNFT := "./cadence-transactions/exampleNFT/mint_exampleNFT.cdc"
	mintExampleNFTCode := util.ParseCadenceTemplate(mintExampleNFT)
	for i := 0; i < 5; i++ {
		_, err := g.TransactionFromFile(mintExampleNFT, mintExampleNFTCode).
			SignProposeAndPayAs("issuer").
			AccountArgument("issuer").
			RunE()
		if err != nil {
			t.Fatal(err)
		}
	}

	balanceExampleNFT := "./cadence-scripts/exampleNFT/balance_exampleNFT.cdc"
	balanceExampleNFTCode := util.ParseCadenceTemplate(balanceExampleNFT)
	nftIDs, err := g.ScriptFromFile(balanceExampleNFT, balanceExampleNFTCode).
		AccountArgument("issuer").RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	arr, ok := nftIDs.(cadence.Array)
	if !ok {
		t.Fatal("can not convert")
	}
	collection := make(common.FlowIDList, len(arr.Values))
	for i := 0; i < len(arr.Values); i++ {
		v, ok := arr.Values[i].(cadence.UInt64)
		if !ok {
			t.Fatal("can not convert 2")
		}
		collection[i] = common.FlowID{Int64: int64(v), Valid: true}
	}

	// PDS share DistCap to PackIssuer (owned by Issuer)

	setDistCap := "./cadence-transactions/pds/set_pack_issuer_cap.cdc"
	setDistCapCode := util.ParseCadenceTemplate(setDistCap)
	_, err = g.TransactionFromFile(setDistCap, setDistCapCode).
		SignProposeAndPayAs("pds").
		AccountArgument("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Issuer now creates distribution

	pdsDistId := "./cadence-scripts/pds/get_current_dist_id.cdc"
	pdsDistIdCode := util.ParseCadenceTemplate(pdsDistId)
	currentDistId, err := g.ScriptFromFile(pdsDistId, pdsDistIdCode).RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	createDist := "./cadence-transactions/pds/create_distribution.cdc"
	createDistCode := util.ParseCadenceTemplate(createDist)
	// Private path must match the PackNFT contract
	e, err := g.TransactionFromFile(createDist, createDistCode).
		SignProposeAndPayAs("issuer").
		Argument(cadence.Path{Domain: "private", Identifier: "exampleNFTCollectionProvider"}).
		RunE()
	if err != nil {
		t.Fatal(err)
	}
	events := util.ParseTestEvents(e)

	util.NewExpectedPDSEvent("DistributionCreated").AddField("DistId", currentDistId.String()).AssertEqual(t, events[0])

	// -- Use newly minted NFTs to create a distribution as issuer --
	d := app.Distribution{
		DistID: common.FlowID{Int64: int64(1), Valid: true}, // TODO
		Issuer: issuer,
		PackTemplate: app.PackTemplate{
			PackReference: app.AddressLocation{
				Name:    "PackNFT",
				Address: issuer,
			},
			PackCount: 2,
			Buckets: []app.Bucket{
				{
					CollectibleReference: app.AddressLocation{
						Name:    "ExampleNFT",
						Address: issuer,
					},
					CollectibleCount:      2,
					CollectibleCollection: collection,
				},
			},
		},
	}

	if err := a.CreateDistribution(context.Background(), &d); err != nil {
		t.Fatal(err)
	}

	// -- Settle --

	// setup examplenft collection for pds (placeholder escrow)
	setupExampleNFT := "./cadence-transactions/exampleNFT/setup_exampleNFT.cdc"
	setupExampleNFTCode := util.ParseCadenceTemplate(setupExampleNFT)
	_, err = g.TransactionFromFile(setupExampleNFT, setupExampleNFTCode).
		SignProposeAndPayAs("pds").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the distribution to go into "settling" state
	for {
		// TODO (latenssi): timeout
		dist, _, err := a.GetDistribution(context.Background(), d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if dist.State == common.DistributionStateSettling {
			break
		}
		time.Sleep(time.Second)
	}

	// Wait for settlement
	wg := &sync.WaitGroup{}
	var waitError error
	wg.Add(1)
	go func() {
		// TODO (latenssi): timeout
		for {
			dist, _, err := a.GetDistribution(context.Background(), d.ID)
			if err != nil {
				waitError = err
				break
			}
			if dist.State == common.DistributionStateMinting {
				break
			}
			time.Sleep(time.Second)
		}
		wg.Done()
	}()

	// transfer
	// TODO: use PDS contract interface instead of manually transfering
	transferExampleNFT := "./cadence-transactions/pds/settle_exampleNFT.cdc"
	code4 := util.ParseCadenceTemplate(transferExampleNFT)
	var FlowIDs []cadence.Value
	for _, c := range d.ResolvedCollection() {
		FlowIDs = append(FlowIDs, cadence.UInt64(c.FlowID.Int64))
	}

	_, err = g.TransactionFromFile(transferExampleNFT, code4).
		SignProposeAndPayAs("pds").
		UInt64Argument(currentDistId.ToGoValue().(uint64)).
		Argument(cadence.NewArray(FlowIDs)).
		RunE()

	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()
	if waitError != nil {
		t.Fatal(waitError)
	}

	// -- Mint --

	// Start minting pack NFTs as pds using mint cap shared by the issuer (should store nfts righ into issuers collection)

	var commitHashes []cadence.Value
	ch1 := "abcde1234"
	ch2 := "cdefg9876"
	nextNFTId := "./cadence-scripts/packNFT/packNFT_total_supply.cdc"
	nextNFTIdCode := util.ParseCadenceTemplate(nextNFTId)
	nextId, err := g.ScriptFromFile(nextNFTId, nextNFTIdCode).RunReturns()
	if err != nil {
		t.Fatal(err)
	}
	subId := fmt.Sprintf("%d", nextId.ToGoValue().(uint64)+1)

	commitHashes = append(commitHashes, cadence.NewString(ch1))
	commitHashes = append(commitHashes, cadence.NewString(ch2))

	commitHashesArr := cadence.NewArray(commitHashes)

	mintPackNFT := "./cadence-transactions/pds/mint_packNFT.cdc"
	mintPackNFTCode := util.ParseCadenceTemplate(mintPackNFT)
	e, err = g.TransactionFromFile(mintPackNFT, mintPackNFTCode).
		SignProposeAndPayAs("pds").
		UInt64Argument(currentDistId.ToGoValue().(uint64)).
		Argument(commitHashesArr).
		AccountArgument("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}
	events = util.ParseTestEvents(e)
	issuerAddr := util.GetAccountAddr(g, "issuer")
	util.NewExpectedPackNFTEvent("Deposit").AddField("id", nextId.String()).AddField("to", issuerAddr).AssertEqual(t, events[0])
	util.NewExpectedPackNFTEvent("Mint").AddField("id", nextId.String()).AddField("commitHash", ch1).AssertEqual(t, events[1])
	util.NewExpectedPackNFTEvent("Deposit").AddField("id", subId).AddField("to", issuerAddr).AssertEqual(t, events[2])
	util.NewExpectedPackNFTEvent("Mint").AddField("id", subId).AddField("commitHash", ch2).AssertEqual(t, events[3])

	getCommitHash := "./cadence-scripts/packNFT/packNFT_commitHash.cdc"
	getCommitHashCode := util.ParseCadenceTemplate(getCommitHash)
	returnedCommitHash, err := g.ScriptFromFile(getCommitHash, getCommitHashCode).AccountArgument("issuer").Argument(nextId).RunReturns()
	if err != nil {
		t.Fatal(err)
	}

	r := returnedCommitHash.ToGoValue().(string)
	if r != ch1 {
		t.Fatal()
	}

	// Wait for minting to finish

	// Issuer transfer PackNFT to owner
	transferPackNFT := "./cadence-transactions/packNFT/transfer_packNFT.cdc"
	transferPackNFTCode := util.ParseCadenceTemplate(transferPackNFT)
	_, err = g.TransactionFromFile(transferPackNFT, transferPackNFTCode).
		SignProposeAndPayAs("issuer").
		AccountArgument("owner").
		Argument(nextId).
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	// -- Reveal --
	// Owner requests to reveal PackNFT

	reveal := "./cadence-transactions/packNFT/reveal.cdc"
	revealCode := util.ParseCadenceTemplate(reveal)
	e, err = g.TransactionFromFile(reveal, revealCode).
		SignProposeAndPayAs("owner").
		Argument(nextId).
		RunE()
	if err != nil {
		t.Fatal(err)
	}
	events = util.ParseTestEvents(e)
	ownerAddr := util.GetAccountAddr(g, "owner")
	util.NewExpectedPackNFTEvent("Withdraw").AddField("id", nextId.String()).AddField("from", ownerAddr).AssertEqual(t, events[0])
	util.NewExpectedPackNFTEvent("RevealRequest").AddField("id", nextId.String()).AssertEqual(t, events[1])
	util.NewExpectedPackNFTEvent("Deposit").AddField("id", nextId.String()).AddField("to", ownerAddr).AssertEqual(t, events[2])

	// -- Retrieve --
}
