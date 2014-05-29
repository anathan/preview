name             'preview'
maintainer       'Nick Gerakines'
maintainer_email 'nick@gerakines'
license          'MIT'
description      'Installs/Configures preview'
long_description IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version          '0.2.2'

depends 'apt'
depends 'yum'
depends 'yum-epel'
depends 'ark'
depends 'monit', '~> 1.5.3'
depends 'build-essential', '~> 2.0'

supports 'centos'
