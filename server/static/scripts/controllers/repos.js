(function () {

	/**
	 * ReposCtrl responsible for rendering the user's
	 * repository home screen.
	 */	
	function ReposCtrl($scope, $routeParams, repos, users) {

		// Gets the currently authenticated user 
		users.getCached().then(function(payload){
			$scope.user = payload.data;
		});
		
		// Gets a list of repos to display in the
		// dropdown.
		repos.list().then(function(payload){
			$scope.repos = angular.isArray(payload.data) ? payload.data : [];
		}).catch(function(err){
			$scope.error = err;
		});
	}

	/**
	 * RepoAddCtrl responsible for activaing a new
	 * repository.
	 */	
	function RepoAddCtrl($scope, $location, repos, users) {
		$scope.add = function(slug) {
			repos.post(slug).then(function(payload) {
				$location.path('/'+slug);
			}).catch(function(err){
				$scope.error = err;
			});
		}
	}

	/**
	 * RepoEditCtrl responsible for editing a repository.
	 */	
	function RepoEditCtrl($scope, $location, $routeParams, repos, users) {
		var owner = $routeParams.owner;
		var name  = $routeParams.name;
		var fullName = owner+'/'+name;

		// Gets the currently authenticated user 
		users.getCached().then(function(payload){
			$scope.user = payload.data;
		});
		
		// Gets a repository
		repos.get(fullName).then(function(payload){
			$scope.repo = payload.data;
		}).catch(function(err){
			$scope.error = err;
		});

		$scope.save = function(repo) {
			repos.update(repo).then(function(payload) {
				$scope.repo = payload.data;
			}).catch(function(err){
				$scope.error = err;
			});
		}

		$scope.delete = function(repo) {
			repos.delete(repo).then(function(payload) {
				$location.path('/');
			}).catch(function(err){
				$scope.error = err;
			});
		}

		$scope.param={}
		$scope.addParam = function(param) {
			if (!$scope.repo.params) {
				$scope.repo.params = {}
			}
			$scope.repo.params[param.key]=param.value;
			$scope.param={}
		}

		$scope.deleteParam = function(key) {
			delete $scope.repo.params[key];
		}
	}

	angular
		.module('drone')
		.controller('ReposCtrl', ReposCtrl)
		.controller('RepoAddCtrl', RepoAddCtrl)
		.controller('RepoEditCtrl', RepoEditCtrl);
})();