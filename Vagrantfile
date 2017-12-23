# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/xenial64"
  config.vm.network "public_network"
  config.vm.network "forwarded_port", guest: 8080, host: 8080
  config.vm.provision "shell", inline: <<-SHELL
    apt -y update
    apt -y install python3 python3-dev python3-pip curl avahi-utils

    pip3 install --upgrade pip

    cd /vagrant
    pip3 install -r requirements.txt
  SHELL
end
