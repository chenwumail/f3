# f3

## install
./f3
it will create ./files directory to save files and listen on port 80

## example
  test filename：test.tar.gz

### upload
curl localhost -F f=@test.tar.gz

### download 
curl -O localhost/test.tar.gz
