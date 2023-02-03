# Create pollinate DB
create database pollinate;
create user pollinate with encrypted password 'pollinate';
grant all privileges on database pollinate to pollinate;
