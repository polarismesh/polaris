#!/bin/bash

repository="repository"
cd frontend
go build main.go
docker build -t $repository/outlier_frontend .

cd ../backend
go build main.go
docker build -t $repository/outlier_backend .

docker push $repository/outlier_frontend
docker push $repository/outlier_backend

cd ../
sed -i "s/repository/$repository/g" ./outlier.yaml

