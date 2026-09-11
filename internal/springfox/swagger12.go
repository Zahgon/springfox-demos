package springfox

import "sort"

// renderSwagger12 builds the Swagger 1.2 resource listing served at /api-docs
// when @EnableSwagger is present.
//
// The 1.2 writer differs from the 2.0 and 3.0 ones in three visible ways: it
// emits a resource *listing* rather than an API declaration, so its `apis`
// array names one entry per controller tag rather than per path; it serialises
// its keys in alphabetical order; and it flattens `contact` to the contact's
// name.
func (p *Provider) renderSwagger12(g *group) any {
	return obj().
		set("apiVersion", g.docket.Info().Version).
		set("apis", swagger12APIs(g)).
		set("authorizations", swagger12Authorizations(g.docket.securitySchemes)).
		set("info", swagger12Info(g.docket.Info())).
		set("swaggerVersion", "1.2")
}

// swagger12APIs lists the group's controller tags as sub-resources, at
// "/<group>/<tag>", sorted by path.
func swagger12APIs(g *group) []*object {
	descriptions := map[string]string{}
	var tags []string
	for _, dop := range g.ops {
		name := dop.op.ListedTag
		if name == "" {
			name = dop.op.Tag
		}
		if name == "" {
			continue
		}
		if _, seen := descriptions[name]; seen {
			continue
		}
		description := dop.op.ListedTagDescription
		if description == "" {
			description = humanise(name)
		}
		descriptions[name] = description
		tags = append(tags, name)
	}
	sort.Strings(tags)

	out := make([]*object, 0, len(tags))
	for _, tag := range tags {
		out = append(out, obj().
			set("description", descriptions[tag]).
			set("path", "/"+g.docket.Group()+"/"+tag).
			set("position", 0))
	}
	return out
}

// swagger12Info flattens the ApiInfo: `contact` is the contact's name rather
// than an object — and is written even when empty — and there is no version
// field, because apiVersion carries it.
func swagger12Info(info APIInfo) *object {
	return obj().
		set("contact", info.Contact.Name).
		set("description", info.Description).
		setNonEmpty("license", info.License).
		setNonEmpty("licenseUrl", info.LicenseURL).
		set("termsOfServiceUrl", info.TermsOfServiceURL).
		set("title", info.Title)
}

// swagger12Authorizations renders the Docket's schemes keyed by their 1.2 type
// name, which is "oauth2" or "basicAuth" rather than the scheme's own name.
func swagger12Authorizations(schemes []SecurityScheme) *object {
	out := obj()
	for _, s := range schemes {
		switch s.Type {
		case SchemeOAuth2:
			grantTypes := obj()
			for _, gt := range s.GrantTypes {
				grantTypes.set(gt.Type, obj().
					set("loginEndpoint", obj().set("url", gt.LoginEndpoint)).
					set("type", gt.Type))
			}
			scopes := make([]*object, 0, len(s.Scopes))
			for _, sc := range s.Scopes {
				scopes = append(scopes, obj().
					set("description", sc.Description).
					set("scope", sc.Scope))
			}
			out.set("oauth2", obj().
				set("grantTypes", grantTypes).
				set("name", s.Name).
				set("scopes", scopes).
				set("type", "oauth2"))
		case SchemeBasic:
			out.set("basicAuth", obj().
				set("name", s.Name).
				set("type", "basicAuth"))
		case SchemeAPIKey:
			out.set("apiKey", obj().
				set("keyname", s.KeyName).
				set("name", s.Name).
				set("passAs", s.PassAs).
				set("type", "apiKey"))
		}
	}
	return out
}
