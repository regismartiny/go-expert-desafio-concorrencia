package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
)

type AuctionRepositoryTestSuite struct {
	suite.Suite
	context  context.Context
	DbServer *mim.Server
	Db       *mongo.Database
}

func (suite *AuctionRepositoryTestSuite) SetupSuite() {
	testCtx := context.Background()
	suite.context = testCtx

	server, err := mim.Start(testCtx, "6.0.4")

	if err != nil {
		log.Fatal(err.Error())
	}
	suite.NoError(err)
	suite.DbServer = server

	client, err := mongo.Connect(testCtx, options.Client().ApplyURI(server.URI()))
	suite.NoError(err)

	if err := client.Ping(testCtx, nil); err != nil {
		logger.Error("Error trying to ping test mongodb database", err)
	}
	suite.NoError(err)

	suite.Db = client.Database("testDb")
}

func (suite *AuctionRepositoryTestSuite) TearDownSuite() {
	suite.DbServer.Stop(suite.context)
}

func (suite *AuctionRepositoryTestSuite) TearDownTest() {
	err := suite.Db.Drop(suite.context)
	suite.NoError(err)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(AuctionRepositoryTestSuite))
}

func (suite *AuctionRepositoryTestSuite) TestGivenAnAction_WhenSave_ThenShouldSaveAuction() {

	os.Setenv("AUCTION_INTERVAL", "3s")

	repo := NewAuctionRepository(suite.Db)

	auctionEntity, err := auction_entity.CreateAuction(
		"Produto1",
		"Categoria",
		"Descrição",
		auction_entity.ProductCondition(1))

	suite.Nil(err)

	_ = repo.CreateAuction(suite.context, auctionEntity)

	auctions, _ := repo.FindAuctions(suite.context, 0, "", "")

	suite.Equal(1, len(auctions))
	suite.Equal(auctionEntity.ProductName, auctions[0].ProductName)
	suite.Equal(auctionEntity.Category, auctions[0].Category)
	suite.Equal(auctionEntity.Description, auctions[0].Description)
	suite.Equal(auction_entity.New, auctions[0].Condition)
	suite.Equal(auction_entity.Active, auctions[0].Status)
}

func (suite *AuctionRepositoryTestSuite) TestGivenAnAction_WhenSave_ThenShouldSaveAuctionAndNotUpdateStatusToCompletedBeforeExpiration() {

	os.Setenv("AUCTION_INTERVAL", "3s")

	repo := NewAuctionRepository(suite.Db)

	auctionEntity, err := auction_entity.CreateAuction(
		"Produto1",
		"Categoria",
		"Descrição",
		auction_entity.ProductCondition(1))

	suite.Nil(err)

	_ = repo.CreateAuction(suite.context, auctionEntity)

	time.Sleep(1 * time.Second)

	auctions, _ := repo.FindAuctions(suite.context, auction_entity.Completed, "", "")

	suite.Empty(auctions)
}

func (suite *AuctionRepositoryTestSuite) TestGivenAnAction_WhenSave_ThenShouldSaveAuctionAndUpdateStatusToCompletedAfterExpiration() {

	os.Setenv("AUCTION_INTERVAL", "1s")

	repo := NewAuctionRepository(suite.Db)

	auctionEntity, err := auction_entity.CreateAuction(
		"Produto1",
		"Categoria",
		"Descrição",
		auction_entity.ProductCondition(1))

	suite.Nil(err)

	_ = repo.CreateAuction(suite.context, auctionEntity)

	time.Sleep(5 * time.Second)

	auctions, _ := repo.FindAuctions(suite.context, auction_entity.Completed, "", "")

	suite.Equal(1, len(auctions))
	suite.Equal(auctionEntity.ProductName, auctions[0].ProductName)
	suite.Equal(auctionEntity.Category, auctions[0].Category)
	suite.Equal(auctionEntity.Description, auctions[0].Description)
	suite.Equal(auction_entity.New, auctions[0].Condition)
	suite.Equal(auction_entity.Completed, auctions[0].Status)
}
