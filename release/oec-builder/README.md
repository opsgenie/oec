Example build command:

docker build \
    -t oec-builder \
    --build-arg GO_VERSION=1.12.1 \
    --no-cache .

Example run command:

docker run \
 --entrypoint /input/build \
 -e OEC_VERSION=1.0.3 \
 -e OEC_REPO=/oec_repo \
 -e OUTPUT=/oec_repo/release/oec-builder \
 -v /Users/faziletozer/go/src/github.com/opsgenie/oec:/oec_repo \
 -v $(pwd):/input \
 oec-builder

Run docker run command in oec-linux, oec-win32 and oec-win64 folders, the executables will be generated under oec-packages folder.