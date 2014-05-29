name             'preview_test'
maintainer       'Nick Gerakines'
maintainer_email 'nick@gerakines.net'
license          'MIT'
description      'Installs/Configures preview_test'
long_description IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version          '0.1.0'

depends 'yum'
depends 'yum-epel'
depends 'java'
depends 'cassandra', '~> 2.2.0'
depends 'monit', '~> 1.5.3'
depends 'preview_build'

supports 'centos'
