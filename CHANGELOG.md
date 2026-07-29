# Changelog

## [0.5.0](https://github.com/sakamichi-blog-archive/sba-showroom/compare/v0.4.1...v0.5.0) (2026-07-29)


### Features

* Add check task that runs fmt:check, lint, and test in parallel ([7c6cdd0](https://github.com/sakamichi-blog-archive/sba-showroom/commit/7c6cdd04d7878a63e7b4594b2c7e68e6abf79112))
* Add check-workflows CI job with actionlint, zizmor, and pinact ([d4fb686](https://github.com/sakamichi-blog-archive/sba-showroom/commit/d4fb6867628b5fb042624256dcf9a764d0974ce5))
* Add check:workflows task, remove standalone actionlint and zizmor tasks ([a86d47f](https://github.com/sakamichi-blog-archive/sba-showroom/commit/a86d47f878a26b006c54b22226657c097e541054))
* Add Go implementation with download sub-command for SHOWROOM ([31bcfda](https://github.com/sakamichi-blog-archive/sba-showroom/commit/31bcfdae638b2e23adcc7b6ed3b9955cc4107df9))
* Add help and version commands and flags ([#38](https://github.com/sakamichi-blog-archive/sba-showroom/issues/38)) ([f65b44c](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f65b44cf8a45a2ce3adbb5e0d997728854d30ae0))
* Add pinact and pinact:update mise tasks ([265c781](https://github.com/sakamichi-blog-archive/sba-showroom/commit/265c781c96e854432467d9ffc5873a932bbd820e))
* Add pinact:check task and include it in check ([d759caa](https://github.com/sakamichi-blog-archive/sba-showroom/commit/d759caab961cc9b9149d8c463009fc82df0b3858))
* Add release-please and publish workflows ([79049a5](https://github.com/sakamichi-blog-archive/sba-showroom/commit/79049a5a8b2bb123813e50527d179db920eab109))
* Add shellcheck so actionlint uses it for shell script checks ([e3dc7a9](https://github.com/sakamichi-blog-archive/sba-showroom/commit/e3dc7a9c8ada39abefff2ec588a6717521fdf25e))
* add start/end timestamps to watcher download output ([#26](https://github.com/sakamichi-blog-archive/sba-showroom/issues/26)) ([d60dcc6](https://github.com/sakamichi-blog-archive/sba-showroom/commit/d60dcc653cdc6ee1b3e58085deeb321432aa573f))
* Add zizmor config with dependabot cooldown set to 3 days ([81aa613](https://github.com/sakamichi-blog-archive/sba-showroom/commit/81aa613f02e05810a81317d7b3c72471818ffe73))
* Check stream URLs directly after scheduled time ([#17](https://github.com/sakamichi-blog-archive/sba-showroom/issues/17)) ([9fc762e](https://github.com/sakamichi-blog-archive/sba-showroom/commit/9fc762ef9e66e074c1f306f4a555f3b24ecbbec5))
* Configure release-please for v0.x versioning ([e92a37b](https://github.com/sakamichi-blog-archive/sba-showroom/commit/e92a37b75a0d5d85ae280ed74e53d1eac3e35211))
* Hardcode JST for output filename date ([#7](https://github.com/sakamichi-blog-archive/sba-showroom/issues/7)) ([8eb340e](https://github.com/sakamichi-blog-archive/sba-showroom/commit/8eb340e7a7e621a5a41cd33fb42e90060fb2a32d))
* watch command ([#15](https://github.com/sakamichi-blog-archive/sba-showroom/issues/15)) ([1868bbd](https://github.com/sakamichi-blog-archive/sba-showroom/commit/1868bbda1323f43fee72c021c5f4492c650d311e))


### Bug fixes

* Add issues: write permission to release-please job ([53950ab](https://github.com/sakamichi-blog-archive/sba-showroom/commit/53950ab0da2d6db6ee33e4fd898eada71d4567f0))
* Apply sba-stream behavior ([#18](https://github.com/sakamichi-blog-archive/sba-showroom/issues/18)) ([c3329c7](https://github.com/sakamichi-blog-archive/sba-showroom/commit/c3329c7643962d02ff460e1f743417005d3ae4c8))
* create releases as drafts and publish after assets attach ([#39](https://github.com/sakamichi-blog-archive/sba-showroom/issues/39)) ([2935c37](https://github.com/sakamichi-blog-archive/sba-showroom/commit/2935c3745dd09fcf115a87b520e2fddbee340c91))
* exit 0 if download produced a file, even on ffmpeg error ([#12](https://github.com/sakamichi-blog-archive/sba-showroom/issues/12)) ([f67e953](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f67e95307abcb9612a82304d3cdbe727a3e450d7))
* Fix duplicate download passthrough ([#27](https://github.com/sakamichi-blog-archive/sba-showroom/issues/27)) ([7e67529](https://github.com/sakamichi-blog-archive/sba-showroom/commit/7e6752904685acf6fdfbe0f8abd25cbda5ac7ac5))
* Fix watcher concurrent duplicate recording ([#28](https://github.com/sakamichi-blog-archive/sba-showroom/issues/28)) ([7847660](https://github.com/sakamichi-blog-archive/sba-showroom/commit/7847660a7c4c02f3b3e4ff1bee32d780a4d75271))
* Fix watcher duplicate recording for aliased room keys ([#29](https://github.com/sakamichi-blog-archive/sba-showroom/issues/29)) ([c1102d9](https://github.com/sakamichi-blog-archive/sba-showroom/commit/c1102d9d6828aee992ba41d2e370d9b62f5a3d76))
* Hoist matrix and release vars to job-level env ([3d7aeb7](https://github.com/sakamichi-blog-archive/sba-showroom/commit/3d7aeb7153f8169a213a65aa9a42cf1481defbb6))
* Include version tag in release asset names ([77cc62b](https://github.com/sakamichi-blog-archive/sba-showroom/commit/77cc62b33d030f5da517b8bf01c06decf8952860))
* Move bump-minor-pre-major into package config ([f30c775](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f30c775a6c398e0f6915ed30214f765829b9beec))
* Poll streaming URLs and download immediately ([#19](https://github.com/sakamichi-blog-archive/sba-showroom/issues/19)) ([f94b71d](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f94b71d4a8446d6afb1bb67b04d01e89d51d934b))
* Trigger publish on tag push instead of release published ([0aa5da6](https://github.com/sakamichi-blog-archive/sba-showroom/commit/0aa5da6492b49a133fd46849e7cf6cdc2253a833))
* trigger release-please on deps commits ([#35](https://github.com/sakamichi-blog-archive/sba-showroom/issues/35)) ([c88b16f](https://github.com/sakamichi-blog-archive/sba-showroom/commit/c88b16f0c74f830187d8cf0a2e9bce602317fd44))
* Use --clobber on release upload for idempotent reruns ([2f9e881](https://github.com/sakamichi-blog-archive/sba-showroom/commit/2f9e881ce347272dac8da0b4d726dabf7481bb31))


### Dependencies

* Bump actions/setup-go from 6.5.0 to 7.0.0 ([#20](https://github.com/sakamichi-blog-archive/sba-showroom/issues/20)) ([ee6bf75](https://github.com/sakamichi-blog-archive/sba-showroom/commit/ee6bf75ea24d3d7d0db63e2019314b078f4c1f74))
* Bump the all-non-major group with 2 updates ([#24](https://github.com/sakamichi-blog-archive/sba-showroom/issues/24)) ([82d68c0](https://github.com/sakamichi-blog-archive/sba-showroom/commit/82d68c046f12f66c77031b9a2a1a40a493168f5f))

## [0.4.1](https://github.com/sakamichi-blog-archive/sba-showroom/compare/v0.4.0...v0.4.1) (2026-07-29)


### Bug fixes

* create releases as drafts and publish after assets attach ([#39](https://github.com/sakamichi-blog-archive/sba-showroom/issues/39)) ([2935c37](https://github.com/sakamichi-blog-archive/sba-showroom/commit/2935c3745dd09fcf115a87b520e2fddbee340c91))

## [0.4.0](https://github.com/sakamichi-blog-archive/sba-showroom/compare/v0.3.0...v0.4.0) (2026-07-29)


### Features

* Add help and version commands and flags ([#38](https://github.com/sakamichi-blog-archive/sba-showroom/issues/38)) ([f65b44c](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f65b44cf8a45a2ce3adbb5e0d997728854d30ae0))


### Bug fixes

* trigger release-please on deps commits ([#35](https://github.com/sakamichi-blog-archive/sba-showroom/issues/35)) ([c88b16f](https://github.com/sakamichi-blog-archive/sba-showroom/commit/c88b16f0c74f830187d8cf0a2e9bce602317fd44))

## [0.3.0](https://github.com/sakamichi-blog-archive/sba-showroom/compare/v0.2.0...v0.3.0) (2026-07-28)


### Features

* add start/end timestamps to watcher download output ([#26](https://github.com/sakamichi-blog-archive/sba-showroom/issues/26)) ([d60dcc6](https://github.com/sakamichi-blog-archive/sba-showroom/commit/d60dcc653cdc6ee1b3e58085deeb321432aa573f))
* Check stream URLs directly after scheduled time ([#17](https://github.com/sakamichi-blog-archive/sba-showroom/issues/17)) ([9fc762e](https://github.com/sakamichi-blog-archive/sba-showroom/commit/9fc762ef9e66e074c1f306f4a555f3b24ecbbec5))
* watch command ([#15](https://github.com/sakamichi-blog-archive/sba-showroom/issues/15)) ([1868bbd](https://github.com/sakamichi-blog-archive/sba-showroom/commit/1868bbda1323f43fee72c021c5f4492c650d311e))


### Bug Fixes

* Apply sba-stream behavior ([#18](https://github.com/sakamichi-blog-archive/sba-showroom/issues/18)) ([c3329c7](https://github.com/sakamichi-blog-archive/sba-showroom/commit/c3329c7643962d02ff460e1f743417005d3ae4c8))
* exit 0 if download produced a file, even on ffmpeg error ([#12](https://github.com/sakamichi-blog-archive/sba-showroom/issues/12)) ([f67e953](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f67e95307abcb9612a82304d3cdbe727a3e450d7))
* Fix duplicate download passthrough ([#27](https://github.com/sakamichi-blog-archive/sba-showroom/issues/27)) ([7e67529](https://github.com/sakamichi-blog-archive/sba-showroom/commit/7e6752904685acf6fdfbe0f8abd25cbda5ac7ac5))
* Fix watcher concurrent duplicate recording ([#28](https://github.com/sakamichi-blog-archive/sba-showroom/issues/28)) ([7847660](https://github.com/sakamichi-blog-archive/sba-showroom/commit/7847660a7c4c02f3b3e4ff1bee32d780a4d75271))
* Fix watcher duplicate recording for aliased room keys ([#29](https://github.com/sakamichi-blog-archive/sba-showroom/issues/29)) ([c1102d9](https://github.com/sakamichi-blog-archive/sba-showroom/commit/c1102d9d6828aee992ba41d2e370d9b62f5a3d76))
* Poll streaming URLs and download immediately ([#19](https://github.com/sakamichi-blog-archive/sba-showroom/issues/19)) ([f94b71d](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f94b71d4a8446d6afb1bb67b04d01e89d51d934b))

## [0.2.0](https://github.com/sakamichi-blog-archive/sba-showroom/compare/v0.1.1...v0.2.0) (2026-07-07)


### Features

* Hardcode JST for output filename date ([#7](https://github.com/sakamichi-blog-archive/sba-showroom/issues/7)) ([8eb340e](https://github.com/sakamichi-blog-archive/sba-showroom/commit/8eb340e7a7e621a5a41cd33fb42e90060fb2a32d))

## [0.1.1](https://github.com/sakamichi-blog-archive/sba-showroom/compare/v0.1.0...v0.1.1) (2026-07-07)


### Bug Fixes

* Trigger publish on tag push instead of release published ([0aa5da6](https://github.com/sakamichi-blog-archive/sba-showroom/commit/0aa5da6492b49a133fd46849e7cf6cdc2253a833))

## [0.1.0](https://github.com/sakamichi-blog-archive/sba-showroom/compare/v0.0.1...v0.1.0) (2026-07-07)


### Features

* Add check task that runs fmt:check, lint, and test in parallel ([7c6cdd0](https://github.com/sakamichi-blog-archive/sba-showroom/commit/7c6cdd04d7878a63e7b4594b2c7e68e6abf79112))
* Add check-workflows CI job with actionlint, zizmor, and pinact ([d4fb686](https://github.com/sakamichi-blog-archive/sba-showroom/commit/d4fb6867628b5fb042624256dcf9a764d0974ce5))
* Add check:workflows task, remove standalone actionlint and zizmor tasks ([a86d47f](https://github.com/sakamichi-blog-archive/sba-showroom/commit/a86d47f878a26b006c54b22226657c097e541054))
* Add Go implementation with download sub-command for SHOWROOM ([31bcfda](https://github.com/sakamichi-blog-archive/sba-showroom/commit/31bcfdae638b2e23adcc7b6ed3b9955cc4107df9))
* Add pinact and pinact:update mise tasks ([265c781](https://github.com/sakamichi-blog-archive/sba-showroom/commit/265c781c96e854432467d9ffc5873a932bbd820e))
* Add pinact:check task and include it in check ([d759caa](https://github.com/sakamichi-blog-archive/sba-showroom/commit/d759caab961cc9b9149d8c463009fc82df0b3858))
* Add release-please and publish workflows ([79049a5](https://github.com/sakamichi-blog-archive/sba-showroom/commit/79049a5a8b2bb123813e50527d179db920eab109))
* Add shellcheck so actionlint uses it for shell script checks ([e3dc7a9](https://github.com/sakamichi-blog-archive/sba-showroom/commit/e3dc7a9c8ada39abefff2ec588a6717521fdf25e))
* Add zizmor config with dependabot cooldown set to 3 days ([81aa613](https://github.com/sakamichi-blog-archive/sba-showroom/commit/81aa613f02e05810a81317d7b3c72471818ffe73))
* Configure release-please for v0.x versioning ([e92a37b](https://github.com/sakamichi-blog-archive/sba-showroom/commit/e92a37b75a0d5d85ae280ed74e53d1eac3e35211))


### Bug Fixes

* Add issues: write permission to release-please job ([53950ab](https://github.com/sakamichi-blog-archive/sba-showroom/commit/53950ab0da2d6db6ee33e4fd898eada71d4567f0))
* Hoist matrix and release vars to job-level env ([3d7aeb7](https://github.com/sakamichi-blog-archive/sba-showroom/commit/3d7aeb7153f8169a213a65aa9a42cf1481defbb6))
* Include version tag in release asset names ([77cc62b](https://github.com/sakamichi-blog-archive/sba-showroom/commit/77cc62b33d030f5da517b8bf01c06decf8952860))
* Move bump-minor-pre-major into package config ([f30c775](https://github.com/sakamichi-blog-archive/sba-showroom/commit/f30c775a6c398e0f6915ed30214f765829b9beec))
* Use --clobber on release upload for idempotent reruns ([2f9e881](https://github.com/sakamichi-blog-archive/sba-showroom/commit/2f9e881ce347272dac8da0b4d726dabf7481bb31))
