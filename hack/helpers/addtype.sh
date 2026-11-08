#!/usr/bin/env bash

# Copyright 2026 The Crossplane Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Please set ProviderNameLower & ProviderNameUpper environment variables before running this script.
# See: https://github.com/crossplane/terrajet/blob/main/docs/generating-a-provider.md
set -euo pipefail

APIVERSION="${APIVERSION:-v1alpha1}"
echo "Adding type ${KIND} to group ${GROUP} with version ${APIVERSION}"

export GROUP
export KIND
export APIVERSION
export PROVIDER
export PROJECT_REPO

kind_lower=$(echo "${KIND}" | tr "[:upper:]" "[:lower:]")
group_lower=$(echo "${GROUP}" | tr "[:upper:]" "[:lower:]")

# render_if_absent renders template $1 to $2, unless $2 already exists: the
# group and version files are shared by every kind of an existing group.
render_if_absent() {
  if [ ! -f "$2" ]; then
    ${GOMPLATE} < "$1" > "$2"
  fi
}

mkdir -p "apis/${group_lower}/${APIVERSION}"
render_if_absent "hack/helpers/apis/GROUP_LOWER/GROUP_LOWER.go.tmpl" "apis/${group_lower}/${group_lower}.go"
${GOMPLATE} < "hack/helpers/apis/GROUP_LOWER/APIVERSION/KIND_LOWER_types.go.tmpl" > "apis/${group_lower}/${APIVERSION}/${kind_lower}_types.go"
render_if_absent "hack/helpers/apis/GROUP_LOWER/APIVERSION/doc.go.tmpl" "apis/${group_lower}/${APIVERSION}/doc.go"
render_if_absent "hack/helpers/apis/GROUP_LOWER/APIVERSION/groupversion_info.go.tmpl" "apis/${group_lower}/${APIVERSION}/groupversion_info.go"

mkdir -p "internal/controller/${group_lower}/${kind_lower}"
${GOMPLATE} < "hack/helpers/controller/KIND_LOWER/KIND_LOWER.go.tmpl" > "internal/controller/${group_lower}/${kind_lower}/${kind_lower}.go"
${GOMPLATE} < "hack/helpers/controller/KIND_LOWER/KIND_LOWER_test.go.tmpl" > "internal/controller/${group_lower}/${kind_lower}/${kind_lower}_test.go"

mkdir -p "internal/clients/${group_lower}"
# Skip file if it already exists to avoid overwriting any custom code added by the user.
if [ ! -f "internal/clients/${group_lower}/${kind_lower}.go" ]; then
  ${GOMPLATE} < "hack/helpers/clients/GROUP_LOWER/KIND_LOWER.go.tmpl" > "internal/clients/${group_lower}/${kind_lower}.go"
else
  echo "Warning: internal/clients/${group_lower}/${kind_lower}.go already exists. Skipping file generation to avoid overwriting any custom code added by the user. Some recommended code may be missing from the file."
fi
# Skip file if it already exists to avoid overwriting any custom code added by the user.
if [ ! -f "internal/clients/${group_lower}/${kind_lower}_test.go" ]; then
  ${GOMPLATE} < "hack/helpers/clients/GROUP_LOWER/KIND_LOWER_test.go.tmpl" > "internal/clients/${group_lower}/${kind_lower}_test.go"
else
  echo "Warning: internal/clients/${group_lower}/${kind_lower}_test.go already exists. Skipping file generation to avoid overwriting any custom code added by the user. Some recommended code may be missing from the file."
fi

# The import order of the templates depends on the group name.
gofmt -w \
  "apis/${group_lower}/${APIVERSION}/${kind_lower}_types.go" \
  "internal/controller/${group_lower}/${kind_lower}" \
  "internal/clients/${group_lower}/${kind_lower}.go" \
  "internal/clients/${group_lower}/${kind_lower}_test.go"

echo "Register the controller in the managedControllers list of internal/controller/register.go, then run 'make generate' to update the generated code with the new type."
