'use strict';

angular.module('app').filter('gravatar', function() {
  return function(gravatar) {
    return "https://secure.gravatar.com/avatar/"+gravatar+"?s=32&d=mm"
  }
});

angular.module('app').filter('fromNow', function() {
  return function(date) {
    return moment(new Date(date*1000)).fromNow();
  }
});

angular.module('app').filter('toDuration', function() {
  return function(seconds) {
  	return moment.duration(seconds, "seconds").humanize();
  }
});

angular.module('app').filter('toDate', function() {
  return function(date) {
    return moment(new Date(date*1000)).format('ll');
  }
});

angular.module('app').filter('fullName', function() {
  return function(repo) {
    return repo.owner+"/"+repo.name;
  }
});

angular.module('app').filter('fullPath', function() {
  return function(repo) {
    return repo.remote+"/"+repo.owner+"/"+repo.name;
  }
});

angular.module('app').filter('shortHash', function() {
  return function(sha) {
  	if (!sha) { return ""; }
    return sha.substr(0,10)
  }
});

angular.module('app').filter('badgeMarkdown', function() {
  return function(repo) {
    var scheme = window.location.protocol;
    var host = window.location.host;
    var path = repo.host+'/'+repo.owner+'/'+repo.name;
    return '[![Build Status]('+scheme+'//'+host+'/v1/badge/'+path+'/status.svg?branch=master)]('+scheme+'//'+host+'/'+path+')'
  }
});

angular.module('app').filter('badgeMarkup', function() {
  return function(repo) {
    var scheme = window.location.protocol;
    var host = window.location.host;
    var path = repo.host+'/'+repo.owner+'/'+repo.name;
    return '<a href="'+scheme+'//'+host+'/'+path+'"><img src="'+scheme+'//'+host+'/v1/badge/'+path+'/status.svg?branch=master" /></a>'
  }
});