# f3

## install
`code
mkdir /usr/local/f3
cp f3 /usr/local/f3/
cp f3.service /etc/systemd/system/f3.service
systemctl daemon-reload
systemctl start f3
it will create ./files directory to save files and listen on port 80
`

## example
  test filename：test.tar.gz

### upload
curl localhost -T test.tar.gz

### download 
curl localhost/test.tar.gz -O
