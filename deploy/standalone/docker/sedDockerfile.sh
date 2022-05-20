version=$1
sed -i "1s/polaris-server-tag/${version}/" ./Dockerfile