// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	spec "github.com/drone/drone/cmd/drone-server/config"
	"github.com/drone/drone/core"
	"github.com/drone/drone/plugin/admission"
	"github.com/drone/drone/plugin/config"
	"github.com/drone/drone/plugin/converter"
	"github.com/drone/drone/plugin/registry"
	"github.com/drone/drone/plugin/secret"
	"github.com/drone/drone/plugin/validator"
	"github.com/drone/drone/plugin/webhook"
	"github.com/drone/go-scm/scm"

	"github.com/google/wire"
)

// wire set for loading plugins.
var pluginSet = wire.NewSet(
	provideAdmissionPlugin,
	provideConfigPlugin,
	provideConvertPlugin,
	provideRegistryPlugin,
	provideSecretPlugin,
	provideValidatePlugin,
	provideWebhookPlugin,
)

// provideAdmissionPlugin is a Wire provider function that
// returns an admission plugin based on the environment
// configuration.
func provideAdmissionPlugin(client *scm.Client, orgs core.OrganizationService, users core.UserService, config spec.Config) core.AdmissionService {
	return admission.Combine(
		admission.Membership(orgs, config.Users.Filter),
		admission.Open(config.Registration.Closed),
		admission.Nobot(users, config.Users.MinAge),
		admission.External(
			config.Authn.Endpoint,
			config.Authn.Secret,
			config.Authn.SkipVerify,
		),
	)
}

// provideConfigPlugin is a Wire provider function that returns
// a yaml configuration plugin based on the environment
// configuration.
func provideConfigPlugin(client *scm.Client, contents core.FileService, conf spec.Config) core.ConfigService {
	return config.Combine(
		config.Memoize(
			config.Global(
				conf.Yaml.Endpoint,
				conf.Yaml.Secret,
				conf.Yaml.SkipVerify,
				conf.Yaml.Timeout,
			),
		),
		config.Repository(contents),
	)
}

// provideConvertPlugin is a Wire provider function that returns
// a yaml conversion plugin based on the environment
// configuration.
func provideConvertPlugin(client *scm.Client, conf spec.Config, templateStore core.TemplateStore) core.ConvertService {
	return converter.Combine(
		converter.Legacy(false),
		converter.Starlark(
			conf.Starlark.Enabled,
		),
		converter.Jsonnet(
			conf.Jsonnet.Enabled,
			client,
		),
		converter.Template(
			templateStore,
		),
		converter.Memoize(
			converter.Remote(
				conf.Convert.Endpoint,
				conf.Convert.Secret,
				conf.Convert.Extension,
				conf.Convert.SkipVerify,
				conf.Convert.Timeout,
			),
			conf.Convert.CacheSize,
		),
	)
}

// provideRegistryPlugin is a Wire provider function that
// returns a registry plugin based on the environment
// configuration.
func provideRegistryPlugin(config spec.Config) core.RegistryService {
	return registry.Combine(
		registry.External(
			config.Secrets.Endpoint,
			config.Secrets.Password,
			config.Secrets.SkipVerify,
		),
		registry.FileSource(
			config.Docker.Config,
		),
		registry.EndpointSource(
			config.Registries.Endpoint,
			config.Registries.Password,
			config.Registries.SkipVerify,
		),
	)
}

// provideSecretPlugin is a Wire provider function that returns
// a secret plugin based on the environment configuration.
func provideSecretPlugin(config spec.Config) core.SecretService {
	return secret.External(
		config.Secrets.Endpoint,
		config.Secrets.Password,
		config.Secrets.SkipVerify,
	)
}

// provideValidatePlugin is a Wire provider function that
// returns a yaml validation plugin based on the environment
// configuration.
func provideValidatePlugin(conf spec.Config) core.ValidateService {
	return validator.Combine(
		validator.Remote(
			conf.Validate.Endpoint,
			conf.Validate.Secret,
			conf.Validate.SkipVerify,
			conf.Validate.Timeout,
		),
		// THIS FEATURE IS INTERNAL USE ONLY AND SHOULD
		// NOT BE USED OR RELIED UPON IN PRODUCTION.
		validator.Filter(
			nil,
			conf.Repository.Ignore,
		),
	)
}

// provideWebhookPlugin is a Wire provider function that returns
// a webhook plugin based on the environment configuration.
func provideWebhookPlugin(config spec.Config, system *core.System) core.WebhookSender {
	return webhook.New(webhook.Config{
		Events:   config.Webhook.Events,
		Endpoint: config.Webhook.Endpoint,
		Secret:   config.Webhook.Secret,
		System:   system,
	})
}
