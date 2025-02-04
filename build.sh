mkdir build 2> /dev/null
cp .env.release build/.env

# Windows x64
echo "Windows..."
export GOOS=windows
export GOARCH=amd64

package=kodik_anime_dl
output_dir="build"
output_file_name="$output_dir/kodik_anime_dl.exe"

go build -o $output_file_name -ldflags "-s -w" $package
echo "UPX..."
upx $output_file_name > /dev/null


# Linux x64
echo "Linux..."
export GOOS=linux
export GOARCH=amd64

package=kodik_anime_dl
output_dir="build"
output_file_name="$output_dir/kodik_anime_dl"

go build -o $output_file_name -ldflags "-s -w" $package
echo "UPX..."
upx $output_file_name > /dev/null
