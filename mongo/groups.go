package mongogroups

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jansemmelink/groups"
	"github.com/jansemmelink/log"
	"github.com/satori/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

//Groups create a collection of groups stored in mongo
func Groups(mongoURI string, dbName string) (groups.IGroups, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, log.Wrapf(err, "Failed to create mongo client to %s", mongoURI)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return nil, log.Wrapf(err, "Failed to connect to mongo %s", mongoURI)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, log.Wrapf(err, "Failed to check mongo %s", mongoURI)
	}

	collection := client.Database(dbName).Collection("groups")
	return &mongoGroups{
		collection: collection,
	}, nil
} //Groups()

type mongoGroups struct {
	collection *mongo.Collection
}

func (m mongoGroups) List(filter map[string]interface{}, sizeLimit int, orderBy []string) []groups.Group {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	validFieldNames := []string{"name"}
	f := bson.M{}
	if filter != nil {
		for n, v := range filter {
			for _, fn := range validFieldNames {
				if fn == n {
					f[n] = bson.M{"$regex": fmt.Sprintf("%v", v)}
					fmt.Printf("FILTER: \"%s\":%+v\n", n, f[n])
					break
				}
			}
		}
	}

	o := bson.M{}
	if orderBy != nil {
		for _, n := range orderBy {
			for _, fn := range validFieldNames {
				if fn == n {
					o[n] = -1
					break
				}
			}
		}
	}

	if sizeLimit == 0 {
		sizeLimit = 10
	}
	if sizeLimit > 100 {
		sizeLimit = 100
	}

	opts := options.Find().SetLimit(int64(sizeLimit)).SetSort(o)
	cur, err := m.collection.Find(ctx, f, opts)
	if err != nil {
		log.Errorf("Failed to find groups: %v", err)
		return nil
	}
	defer cur.Close(ctx)

	groupList := []groups.Group{}
	for cur.Next(ctx) {
		var groupData bson.M
		err := cur.Decode(&groupData)
		if err != nil {
			log.Errorf("Failed to get data: %v", err)
			return nil
		}
		g := groups.Group{
			ID:   groupData["id"].(string),
			Name: groupData["name"].(string),
		}
		groupList = append(groupList, g)
	}

	if err := cur.Err(); err != nil {
		log.Errorf("Error: %v", err)
		return nil
	}
	fmt.Printf("Returning %d groups:\n", len(groupList))
	for i, g := range groupList {
		fmt.Printf("  %d: %+v\n", i, g)
	}
	return groupList
}

func (m mongoGroups) New(g groups.Group) (groups.Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// //make sure the name is uniq
	// todo: should only be unique in parent group
	// if m.GetName(m) != nil {
	// 	return nil, log.Wrapf(nil, "Group already exists with name \"%s\"", m)
	// }

	g.Name = strings.TrimSpace(g.Name)
	if err := g.Validate(); err != nil {
		return g, log.Wrapf(err, "cannot create invalid group")
	}

	g.ID = uuid.NewV1().String()
	_, err := m.collection.InsertOne(
		ctx,
		bson.M{
			"id":   g.ID,
			"name": g.Name,
		})
	if err != nil {
		return g, log.Wrapf(err, "failed to insert group into db")
	}
	return g, nil
} //mongoGroups.New()

func (m mongoGroups) Get(id string) (*groups.Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cur, err := m.collection.Find(ctx, bson.M{"id": id})
	if err != nil {
		return nil, fmt.Errorf("group not found: %v", err)
	}
	defer cur.Close(ctx)

	//for expexts only one result
	for cur.Next(ctx) {
		var result bson.M
		err := cur.Decode(&result)
		if err != nil {
			return nil, fmt.Errorf("group data error: %v", err)
		}
		// do something with result....
		log.Debugf("GOT (%T): %+v", result, result)
		return &groups.Group{
			ID:   result["id"].(string),
			Name: result["name"].(string),
		}, nil
	}

	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("failed to get group data: %v", err)
	}

	//not found
	return nil, nil
} //mongoGroups.Get()

func (m mongoGroups) Upd(g groups.Group) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := m.collection.UpdateOne(ctx,
		bson.M{
			"id": g.ID,
		},
		bson.D{
			{"$set", bson.D{{"name", g.Name}}},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update group %v", err)
	}
	return nil
} //mongoGroups.Upd()

func (m mongoGroups) Del(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := m.collection.FindOneAndDelete(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete group %v", err)
	}
	return nil
} //mongoGroups.Del()
