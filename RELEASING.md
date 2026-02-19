# Releasing

## Overview

This document outlines the release process for the AWS Service Catalog Engine for Terraform Cloud module. This Terraform module enables AWS Service Catalog integration with Terraform Cloud, and follows a structured release process to ensure quality and consistency.

## Versioning

This project follows [Semantic Versioning](https://semver.org/) (SemVer):

- **MAJOR** version (v**X**.0.0): Incompatible API changes
- **MINOR** version (v1.**X**.0): New functionality in a backwards-compatible manner
- **PATCH** version (v1.0.**X**): Backwards-compatible bug fixes

## Release Checklist

Follow these steps to create a new release:

1. **Ensure the engine is working as expected**
   - Provision the engine yourself and attempt to provision the example product
   - Verify that all CI/CD checks are passing
   - Make sure the binaries for each of the lambda functions have been built since the most recent code change

2. **Update CHANGELOG.md**
   - Add a new version entry at the top of the file
   - Use the format: `# v{version} (MM/DD/YYYY)`
   - Include relevant sections:
     - `## Bug Fixes` - For bug fixes and patches
     - `## Improvements` - For new features and enhancements
   - Reference GitHub issues using the format: `[#{issue_number}](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues/{issue_number})`
   - Attribute contributors using: `thanks [@{username}](https://github.com/{username})!`
   - Provide clear, concise descriptions of changes

3. **Commit the changelog update**
   ```bash
   git add CHANGELOG.md
   git commit -m "Update CHANGELOG for v1.0.4 release"
   ```

4. **Create a git tag**
   ```bash
   git tag v1.0.4
   ```

5. **Push the tag to GitHub**
   ```bash
   git push origin v1.0.4
   ```

6. **Create a GitHub Release**
   - Navigate to the [Releases page](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/releases)
   - Click "Draft a new release"
   - Select the tag you just pushed (e.g., `v1.0.4`)
   - Set the release title to match the version (e.g., `v1.0.4`)
   - Copy the relevant section from CHANGELOG.md into the release notes
   - Review and publish the release

## Example

Here's a complete example of releasing version v1.0.4:

### 1. Update CHANGELOG.md

Add the following entry at the top of `CHANGELOG.md`:

```markdown
# v1.0.4 (02/19/2026)

## Bug Fixes
* [#42](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues/42) - Fix provider configuration parsing for aliased providers

## Improvements
* [#45](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues/45) thanks [@contributor](https://github.com/contributor)! - Add support for additional AWS regions
```

### 2. Commit and tag

```bash
# Commit the changelog
git add CHANGELOG.md
git commit -m "Update CHANGELOG for v1.0.4 release"

# Create and push the tag
git tag v1.0.4
git push origin main
git push origin v1.0.4
```

### 3. Create GitHub Release

1. Go to https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/releases/new
2. Choose tag: `v1.0.4`
3. Release title: `v1.0.4`
4. Description:
   ```markdown
   ## Bug Fixes
   * [#42](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues/42) - Fix provider configuration parsing for aliased providers

   ## Improvements
   * [#45](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues/45) thanks [@contributor](https://github.com/contributor)! - Add support for additional AWS regions
   ```
5. Click "Publish release"

## Notes

- Always ensure the main branch is stable before creating a release
- Test the module thoroughly before tagging a new version
- Keep release notes clear and user-focused
- For major releases, consider updating documentation and examples