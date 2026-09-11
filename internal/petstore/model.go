// Package petstore is a Go port of the io.springfox:springfox-petstore:2.10.5
// artifact. Three of the demo modules mount its controllers, and for
// spring-xml-swagger they are the entire API surface, so the artifact's
// behaviour — routes, response bodies, storage semantics and the quirks that
// follow from its incomplete request binding — is part of this repository's
// observable contract and is reproduced here rather than approximated.
package petstore

import "encoding/json"

// Category is springfox.petstore.model.Category.
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Tag is springfox.petstore.model.Tag.
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Pet is springfox.petstore.model.Pet. Its JSON shape follows Jackson's bean
// serialisation of the Java class: fields in declaration order, followed by the
// derived `identifier` property that Identifiable.getIdentifier() contributes.
type Pet struct {
	ID        int64     `json:"id"`
	Category  *Category `json:"category"`
	Name      string    `json:"name"`
	PhotoURLs []string  `json:"photoUrls"`
	Tags      []Tag     `json:"tags"`
	Status    string    `json:"status"`
}

// MarshalJSON appends the derived identifier, as getIdentifier() does.
func (p Pet) MarshalJSON() ([]byte, error) {
	type alias Pet
	return json.Marshal(struct {
		alias
		Identifier int64 `json:"identifier"`
	}{alias(p), p.ID})
}

// UnmarshalJSON restores the empty-list defaults the Java fields are
// initialised with, so a body that omits photoUrls or tags deserialises to an
// empty list rather than null.
func (p *Pet) UnmarshalJSON(data []byte) error {
	type alias Pet
	tmp := alias{PhotoURLs: []string{}, Tags: []Tag{}}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if tmp.PhotoURLs == nil {
		tmp.PhotoURLs = []string{}
	}
	if tmp.Tags == nil {
		tmp.Tags = []Tag{}
	}
	*p = Pet(tmp)
	return nil
}

// Identifier is Identifiable<Long>.getIdentifier().
func (p Pet) Identifier() int64 { return p.ID }

// Order is springfox.petstore.model.Order.
type Order struct {
	ID       int64   `json:"id"`
	PetID    int64   `json:"petId"`
	Quantity int32   `json:"quantity"`
	ShipDate *string `json:"shipDate"`
	Status   *string `json:"status"`
	Complete bool    `json:"complete"`
}

// MarshalJSON appends the derived identifier.
func (o Order) MarshalJSON() ([]byte, error) {
	type alias Order
	return json.Marshal(struct {
		alias
		Identifier int64 `json:"identifier"`
	}{alias(o), o.ID})
}

// Identifier is Identifiable<Long>.getIdentifier().
func (o Order) Identifier() int64 { return o.ID }

// User is springfox.petstore.model.User.
type User struct {
	ID         int64   `json:"id"`
	Username   *string `json:"username"`
	FirstName  *string `json:"firstName"`
	LastName   *string `json:"lastName"`
	Email      *string `json:"email"`
	Password   *string `json:"password"`
	Phone      *string `json:"phone"`
	UserStatus int32   `json:"userStatus"`
}

// MarshalJSON appends the derived identifier, which for a user is its username.
func (u User) MarshalJSON() ([]byte, error) {
	type alias User
	return json.Marshal(struct {
		alias
		Identifier *string `json:"identifier"`
	}{alias(u), u.Username})
}

// Identifier is Identifiable<String>.getIdentifier().
func (u User) Identifier() string {
	if u.Username == nil {
		return ""
	}
	return *u.Username
}

// statusIs is springfox.petstore.model.Pets.statusIs.
func statusIs(status string) func(Pet) bool {
	return func(p Pet) bool { return p.Status == status }
}

// tagsContain is springfox.petstore.model.Pets.tagsContain.
func tagsContain(tag string) func(Pet) bool {
	return func(p Pet) bool {
		for _, t := range p.Tags {
			if t.Name == tag {
				return true
			}
		}
		return false
	}
}
