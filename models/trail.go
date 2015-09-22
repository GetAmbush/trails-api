package models

import (
	"github.com/drborges/appx"
	"time"
	"appengine/datastore"
	"math/rand"
	"appengine"
	"github.com/drborges/geocoder/providers/google"
	"appengine/urlfetch"
	"fmt"
)

type Trail struct {
	appx.Model

	Revision    string               `json:"-"`
	Path        string               `json:"media_path"`
	ThumbExists bool                 `json:"thumb_exists"`

	MimeType    string               `json:"mime_type"`
	CreatedAt   time.Time            `json:"created_at"`
	GeoPoint    appengine.GeoPoint  `json:"geo_point"`
	Tags        []string              `json:"geo_point"`
	Bytes       int64                `json:"bytes"`

	Type        TrailType            `json:"trail_type"`

	Likeness    LikenessType         `json:"likeness"`
	EvaluatedOn time.Time            `json:"evaluated_on"`
}

type LikenessType int
const (
	NotEvaluated    LikenessType = iota
	LikedIt
	DislikedIt
)

type TrailType int
const (
	PhotoTrail    TrailType = iota
	AudioTrail
	VideoTrail
)

func (trail *Trail) KeySpec() *appx.KeySpec {
	return &appx.KeySpec{
		Kind:       "Trails",
		StringID: trail.Revision,
	}
}

func randomDate() time.Time {
	rand.Seed(time.Now().Unix())
	randomMonth := rand.Intn(108)
	rand.Seed(time.Now().Unix() + int64(randomMonth))
	randomDay := rand.Intn(30)

	return time.Now().AddDate(0, -randomMonth, -randomDay)
}

func likeness(trailId string, likeness LikenessType, db *appx.Datastore, context appengine.Context) error {
	trail := Trail{}
	trail.SetEncodedKey(trailId)

	if err := db.Load(&trail); err != nil {
		println("The error: ", err.Error())
		return err
	}

	trail.Likeness = likeness
	trail.EvaluatedOn = time.Now()

	if (trail.Likeness == LikedIt && trail.GeoPoint != appengine.GeoPoint{}) {

		println(">>>>>>About to fetch from Google!")
		trail.Tags = fetchLatLngFromGoogle(trail, context)
	}

	println(fmt.Sprintf("The trail details is: %+v", trail))
	println("")

	if err := db.Save(&trail); err != nil {
		println("The error: ", err.Error())
		return err
	}

	return nil
}

func fetchLatLngFromGoogle(trail Trail, context appengine.Context) []string {
	geoCoder := &google.Geocoder{
		HttpClient:             urlfetch.Client(context),
		ReverseGeocodeEndpoint: google.ReverseGeocodeEndpoint + "&key=AIzaSyC1O6FZtjFDSJz5zCqVbVlVOr60gDYg_Zw",
	}

	res, err := geoCoder.ReverseGeocode(trail.GeoPoint.Lat, trail.GeoPoint.Lng)

	if err != nil { return []string{"Uncategorized"} }

	var address google.Address
	google.ReadResponse(res, &address)

	return []string{address.FullCity, address.FullState, address.FullCountry}
}

var Trails = struct {
	ByNextEvaluation func(account *Account) *datastore.Query
	ByAccount        func(account *Account) *datastore.Query
	Like             func(trailId string, db *appx.Datastore, context appengine.Context) error
	Dislike          func(trailId string, db *appx.Datastore, context appengine.Context) error

}{
	ByNextEvaluation: func(account *Account) *datastore.Query {
		return datastore.NewQuery(new(Trail).KeySpec().Kind).
		Ancestor(account.Key()).
		Filter("CreatedAt >", randomDate()).
		Filter("Likeness =", NotEvaluated).
		Order("CreatedAt").
		Limit(6)
	},

	ByAccount: func(account *Account) *datastore.Query {
		return datastore.NewQuery(new(Trail).KeySpec().Kind).
		Ancestor(account.Key())
	},

	Like: func(trailId string, db *appx.Datastore, context appengine.Context) error {
		return likeness(trailId, LikedIt, db, context)
	},

	Dislike: func(trailId string, db *appx.Datastore, context appengine.Context) error {
		return likeness(trailId, DislikedIt, db, context)
	},
}