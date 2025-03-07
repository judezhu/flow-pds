package pds

import (
	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/onflow/cadence"
)

func CreatePackIssuer(
	g *gwtf.GoWithTheFlow,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txFilename := "../cadence-transactions/pds/create_new_pack_issuer.cdc"
	txScript := util.ParseCadenceTemplate(txFilename)

	e, err := g.TransactionFromFile(txFilename, txScript).
		SignProposeAndPayAs(account).
		RunE()
	events = util.ParseTestEvents(e)
	return
}

func SetPackIssuerCap(
	g *gwtf.GoWithTheFlow,
    issuer string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	setDistCap := "../cadence-transactions/pds/set_pack_issuer_cap.cdc"
	setDistCapCode := util.ParseCadenceTemplate(setDistCap)
	_, err = g.
		TransactionFromFile(setDistCap, setDistCapCode).
		SignProposeAndPayAs("pds").
		AccountArgument("issuer").
		RunE()
    return
}

func CreateDistribution(
	g *gwtf.GoWithTheFlow,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	createDist := "../cadence-transactions/pds/create_distribution.cdc"
	createDistCode := util.ParseCadenceTemplate(createDist)
	// Private path must match the PackNFT contract
	e, err := g.
		TransactionFromFile(createDist, createDistCode).
		SignProposeAndPayAs("issuer").
		Argument(cadence.Path{Domain: "private", Identifier: "exampleNFTCollectionProvider"}).
		RunE()
	events = util.ParseTestEvents(e)
    return
}

func GetDistID(
    g *gwtf.GoWithTheFlow,
) (distId uint64, err error) {
	pdsDistId := "../cadence-scripts/pds/get_current_dist_id.cdc"
	pdsDistIdCode := util.ParseCadenceTemplate(pdsDistId)
    d, err := g.ScriptFromFile(pdsDistId, pdsDistIdCode).RunReturns()
    distId = d.ToGoValue().(uint64)
    return
}

func PDSWithdrawNFT(
	g *gwtf.GoWithTheFlow,
    distId uint64,
    nftIds cadence.Value,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	withdraw := "../cadence-transactions/pds/settle_exampleNFT.cdc"
	withdrawCode := util.ParseCadenceTemplate(withdraw)
    e, err := g.
		TransactionFromFile(withdraw, withdrawCode).
		SignProposeAndPayAs("pds").
        UInt64Argument(distId).
        Argument(nftIds).
		RunE()
	events = util.ParseTestEvents(e)
    return
}

func PDSMintPackNFT(
	g *gwtf.GoWithTheFlow,
    distId uint64,
    commitHash string,
    issuer string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txScript:= "../cadence-transactions/pds/mint_packNFT.cdc"
	code:= util.ParseCadenceTemplate(txScript)
    var arr []cadence.Value
    arr = append(arr, cadence.String(commitHash))
    hashes := cadence.NewArray(arr)
    e, err := g.
		TransactionFromFile(txScript, code).
		SignProposeAndPayAs("pds").
        UInt64Argument(distId).
        Argument(hashes).
        AccountArgument(issuer).
		RunE()
	events = util.ParseTestEvents(e)
    return
}

func PDSRevealPackNFT(
	g *gwtf.GoWithTheFlow,
    distId uint64,
    packId uint64,
    nftContractAddrs cadence.Value,
    nftContractNames cadence.Value,
    nftIds cadence.Value,
    salt string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txScript:= "../cadence-transactions/pds/reveal_packNFT.cdc"
	code:= util.ParseCadenceTemplate(txScript)
    e, err := g.
		TransactionFromFile(txScript, code).
		SignProposeAndPayAs("pds").
        UInt64Argument(distId).
        UInt64Argument(packId).
        Argument(nftContractAddrs).
        Argument(nftContractNames).
        Argument(nftIds).
        StringArgument(salt).
		RunE()
	events = util.ParseTestEvents(e)
    return
}

func PDSOpenPackNFT(
	g *gwtf.GoWithTheFlow,
    distId uint64,
    packId uint64,
    nftIds cadence.Value,
    owner string,
	account string,
) (events []*gwtf.FormatedEvent, err error) {
	txScript:= "../cadence-transactions/pds/open_packNFT.cdc"
	code:= util.ParseCadenceTemplate(txScript)
    e, err := g.
		TransactionFromFile(txScript, code).
		SignProposeAndPayAs("pds").
        UInt64Argument(distId).
        UInt64Argument(packId).
        Argument(nftIds).
        AccountArgument(owner).
		RunE()
	events = util.ParseTestEvents(e)
    return
}
