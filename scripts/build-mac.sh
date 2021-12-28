VERSION="$1"
make clean && make build-app && CGO_ENABLED=1 VERSION=${VERSION} make
FILE_NAME=authorizer-${VERSION}-darwin-amd64.tar.gz
tar cvfz ${FILE_NAME} .env app/build build templates
AUTH="Authorization: token $GITHUB_TOKEN"
RELASE_INFO=$(curl -sH "$AUTH" https://api.github.com/repos/authorizerdev/authorizer/releases/tags/${VERSION})
echo $RELASE_INFO

eval $(echo "$RELASE_INFO" | grep -m 1 "id.:" | grep -w id | tr : = | tr -cd '[[:alnum:]]=')
[ "$id" ] || { echo "Error: Failed to get release id for tag: $VERSION"; echo "$RELASE_INFO" | awk 'length($0)<100' >&2; exit 1; }
echo $id
GH_ASSET="https://uploads.github.com/repos/authorizerdev/authorizer/releases/$id/assets?name=$(basename $FILE_NAME)"

echo $GH_ASSET

curl -H $AUTH -H "Content-Type: $(file -b --mime-type $FILE_NAME)" --data-binary @$FILE_NAME $GH_ASSET

curl "$GITHUB_OAUTH_BASIC" --data-binary @"$FILE_NAME" -H "Authorization: token $GITHUB_TOKEN" -H "Content-Type: application/octet-stream" $GH_ASSET
