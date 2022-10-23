package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	ussd "github.com/jansemmelink/ussd2"
	"github.com/jansemmelink/utils2/errors"
)

func init() {
	ussd.Type("national_id", &NatID{})
	ussd.Type("date", &Date{})

	ussd.Func("profile_menu", profileMenu)
}

type Profile struct {
	OwnerMsisdnList []string `json:"owner_msisdn_list"`
	NatID           NatID    `json:"national_id"`
	Surname         string   `json:"surname"`
	Name            string   `json:"name"`
	Dob             Date     `json:"date"`
	Gender          string   `json:"gender"`
}

func (p Profile) OwnedBy(msisdn string) bool {
	for _, ownerMsisdn := range p.OwnerMsisdnList {
		if msisdn == ownerMsisdn {
			return true
		}
	}
	return false
}

//===== NATIONAL ID NUMBERS =====
type NatID struct {
	value string
}

const natIdPattern = `[0-9]{13}`

var natIdRegex = regexp.MustCompile("^" + natIdPattern + "$")

func (natid NatID) String() string {
	return natid.value
}

func (natid *NatID) Parse(s string) ussd.UserError {
	if len(s) != 13 {
		return ussd.NewUserError("nat_id_not_13_digits", map[string]interface{}{"value": s})
	}
	if !natIdRegex.MatchString(s) {
		return ussd.NewUserError("nat_id_not_13_digits", map[string]interface{}{"value": s})
	}
	natid.value = s
	return nil
}

func (natID NatID) MarshalJSON() ([]byte, error) {
	return []byte("\"" + natID.value + "\""), nil
}

func (natID *NatID) UnmarshalJSON(v []byte) error {
	if ue := natID.Parse(strings.Trim(string(v), "\"")); ue != nil {
		return ue.NewError()
	}
	return nil
}

//===== DATES =====
type Date struct {
	timeValue time.Time
}

func (d Date) String() string {
	return d.timeValue.Format("2006-01-02")
}

func (d *Date) Parse(s string) ussd.UserError {
	var err error
	d.timeValue, err = time.Parse("2006-01-02", s)
	if err != nil {
		return ussd.NewUserError("invalid_date", map[string]interface{}{"value": s, "error": err.Error()})
	}
	return nil
}

func (date Date) MarshalJSON() ([]byte, error) {
	return []byte("\"" + date.String() + "\""), nil
}

func (date *Date) UnmarshalJSON(v []byte) error {
	if ue := date.Parse(strings.Trim(string(v), "\"")); ue != nil {
		return ue.NewError()
	}
	return nil
}

//=====[ PROFILE MENU ]======
type MyProfile struct {
	Name    string
	Profile Profile
}

func itemsList(ids []string) ([]ussd.Item, error) {
	l := []ussd.Item{}
	for _, id := range ids {
		i, ok := ussd.ItemByID(id, nil)
		if !ok {
			return nil, errors.Errorf("unknown item id(%s)", id)
		}
		l = append(l, i)
	}
	return l, nil
}

func profileMenu(ctx context.Context) ([]ussd.Item, error) {
	profileShowItem, ok := ussd.ItemByID("profile_show", nil)
	if !ok {
		return nil, errors.Errorf("missing profile_show item")
	}

	//list of items to call when creating a new profile
	profileNewItemList, err := itemsList([]string{
		"profile_new_natid",
		"fail_if_natid_exists",
		"profile_new_surname",
		"profile_new_name",
		"profile_new_dob",
		"profile_new_gender",
		"profile_add",
	})
	if err != nil {
		return nil, errors.Errorf("cannot create profiles")
	}

	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)
	msisdn := s.Get("msisdn").(string)

	//make list of your existing profiles
	myFamily := []MyProfile{}
	for _, p := range profiles.ProfileByID {
		if p.OwnedBy(msisdn) {
			myFamily = append(myFamily, MyProfile{
				Name:    fmt.Sprintf("%s, %s (%s)", p.Surname, p.Name, p.NatID.String()),
				Profile: p,
			})
		}
	}
	//sort alphabetically
	sort.Slice(myFamily, func(i, j int) bool { return myFamily[i].Name < myFamily[j].Name })

	//start a dynamic menu
	menuDef := ussd.DynMenuDef(ussd.CaptionDef{
		"af": "My Familie"},
	)
	//show list of existing profiles
	for _, p := range myFamily {
		menuDef = menuDef.With(ussd.CaptionDef{"af": p.Name},
			ussd.SetDef{Name: "profile_natid", Value: p.Profile.NatID}.Item(s),
			ussd.SetDef{Name: "profile_surname", Value: p.Profile.Surname}.Item(s),
			ussd.SetDef{Name: "profile_name", Value: p.Profile.Name}.Item(s),
			ussd.SetDef{Name: "profile_dob", Value: p.Profile.Dob}.Item(s),
			ussd.SetDef{Name: "profile_gender", Value: p.Profile.Gender}.Item(s),
			profileShowItem,
		)
	}
	//last option: add a new profile
	menuDef = menuDef.With(ussd.CaptionDef{"af": "Nuwe profiel ..."},
		profileNewItemList...,
	)
	return []ussd.Item{menuDef.Item(s)}, nil
}

func profileAdd(ctx context.Context) ([]ussd.Item, error) {
	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)
	msisdn := s.Get("msisdn").(string)

	p := Profile{}
	if err := p.NatID.Parse(s.Get("profile_new_natid").(string)); err != nil {
		return nil, errors.Wrapf(err.NewError(), "cannot parse nat id")
	}
	if err := p.Dob.Parse(s.Get("profile_new_dob").(string)); err != nil {
		return nil, errors.Wrapf(err.NewError(), "cannot parse dob")
	}
	p.Surname = s.Get("profile_new_surname").(string)
	p.Name = s.Get("profile_new_name").(string)
	p.Gender = s.Get("profile_new_gender").(string)
	p.OwnerMsisdnList = []string{msisdn}

	log.Errorf("New Profile: %+v", p)
	profiles.ProfileByID[p.NatID.String()] = p
	saveProfiles()

	//todo: show profile created, then go back to list of profiles
	return []ussd.Item{ussd.FinalDef{Caption: ussd.CaptionDef{"af": "Die profile is geskep. Totsiens."}}.Item(s)}, nil
}

func failIfNatIDExists(ctx context.Context) ([]ussd.Item, error) {
	s := ctx.Value(ussd.CtxSession{}).(ussd.Session)
	natID := s.Get("profile_new_natid").(NatID)

	profile, ok := profiles.ProfileByID[natID.String()]
	if !ok {
		log.Debugf("NatID does not exist - proceed")
		return nil, nil //does not exist, continue with more prompts
	}

	//nat id already exists
	log.Debugf("NatID already existds with profile: %+v", profile)

	//start a dynamic menu where user can choose from available next actions (added below)
	menuDef := ussd.DynMenuDef(ussd.CaptionDef{"af": "ID {{profile_new_natid}} bestaan alreeds"})

	//option 1: enter another ID
	//first get ussd item to retry the prompt, then add it to the menu as an option
	//todo: ideally should be passed in var so we can use this in other cases too
	promptForOtherNatID, ok := ussd.ItemByID("profile_new_natid", nil)
	if ok {
		menuDef = menuDef.With(ussd.CaptionDef{"af": "Ander ID"},
			promptForOtherNatID,   //prompt again for another id,
			failIfNatIDExistsItem, //then repeat this test
		)
	}

	//option 2: if owned by you, option to edit
	//			if not owned by you, option to invite
	msisdn := s.Get("msisdn").(string)
	if profile.OwnedBy(msisdn) {
		if profileEditItem, ok := ussd.ItemByID("profile_edit", nil); ok {
			menuDef = menuDef.With(ussd.CaptionDef{"af": "Gaan na profiel"},
				// ussd.SetDef{Name: "offer_name", Value: o.Name}.Item(s),
				// ussd.SetDef{Name: "amount", Value: o.Amount}.Item(s),
				profileEditItem,
			)
		}
	} else {
		if inviteNatID, ok := ussd.ItemByID("invite_natid", nil); ok {
			menuDef = menuDef.With(ussd.CaptionDef{"af": "Maak dit ook deel van jou profiel"},
				// ussd.SetDef{Name: "offer_name", Value: o.Name}.Item(s),
				// ussd.SetDef{Name: "amount", Value: o.Amount}.Item(s),
				inviteNatID,
			)
		}
	}

	menuDef = menuDef.With(ussd.CaptionDef{"af": "Ok"},
		ussd.FinalDef{Caption: ussd.CaptionDef{"af": "Totsiens"}}.Item(s),
	)

	//log.Debugf("Defined offers menu: %+v", menuDef)
	return []ussd.Item{menuDef.Item(s)}, nil
}

func init() {
	ussd.Func("profile_add", profileAdd)
	failIfNatIDExistsItem = ussd.Func("fail_if_natid_exists", failIfNatIDExists)
}

var (
	failIfNatIDExistsItem ussd.Item
)

type profileFile struct {
	ProfileByID map[string]Profile `json:"profiles"`
}

var profiles = profileFile{ProfileByID: map[string]Profile{}}

func init() {
	f, err := os.Open("./profiles.json")
	if err == nil {
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&profiles); err != nil {
			panic(fmt.Sprintf("failed to read existing profiles: %+v", err))
		}
	}
}

func saveProfiles() {
	f, err := os.Create("./profiles.json")
	if err != nil {
		panic(fmt.Sprintf("failed to create file for profiles: %+v", err))
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(profiles); err != nil {
		panic(fmt.Sprintf("failed to save profiles: %+v", err))
	}
}
