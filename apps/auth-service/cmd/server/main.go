package main

import authpb "github.com/jsndz/kairo-proto/proto/auth"


type AuthServer struct{
	authpb.UnimplementedAuthServiceServer
}

func main(){

}