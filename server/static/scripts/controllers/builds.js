(function () {

	/**
	 * BuildsCtrl responsible for rendering the repo's
	 * recent build history.
	 */
	function BuildsCtrl($scope, $routeParams, builds, repos, users, feed, logs) {

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

		// Gets a list of builds
		builds.list(fullName).then(function(payload){
			$scope.builds = angular.isArray(payload.data) ? payload.data : [];
		}).catch(function(err){
			$scope.error = err;
		});

		$scope.watch = function(repo) {
			repos.watch(repo.full_name).then(function(payload) {
				$scope.repo.subscription = payload.data;
			});
		}

		$scope.unwatch = function(repo) {
			repos.unwatch(repo.full_name).then(function() {
				delete $scope.repo.subscription;
			});
		}

		feed.subscribe(function(event) {
			if (event.repo.full_name !== fullName) {
				return; // ignore
			}
			// update repository
			$scope.repo = event.repo;
			$scope.$apply();

			var added = false;
			for (var i=0;i<$scope.builds.length;i++) {
				var build = $scope.builds[i];
				if (event.build.number !== build.number) {
					continue; // ignore
				}
				// update the build status
				build.state = event.build.state;
				build.started_at = event.build.started_at;
				build.finished_at = event.build.finished_at;
				build.duration = event.build.duration;
				$scope.builds[i] = build;
				$scope.$apply();
				added = true;
			}

			if (!added) {
				$scope.builds.push(event.build);
				$scope.$apply();
			}
		});
	}

	/**
	 * BuildCtrl responsible for rendering a build.
	 */
	function BuildCtrl($scope, $routeParams, $window, logs, builds, repos, users, feed) {

		var step = parseInt($routeParams.step) || 1;
		var number = $routeParams.number;
		var owner = $routeParams.owner;
		var name  = $routeParams.name;
		var fullName = owner+'/'+name;
		var tail = false;

		// Initiates streaming a build.
		var stream = function() {
			var convert = new Filter({stream:true,newline:false});
			var term = document.getElementById("term");
			term.innerHTML = "";

			// subscribes to the build otuput.
			logs.subscribe(fullName, number, step, function(data){
				term.innerHTML += convert.toHtml(data)+"\n";
				if (tail) {
					// scrolls to the bottom of the page if enabled
					$window.scrollTo(0, $window.document.body.scrollHeight);
				}
			});
		}

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

		// Gets the build
		builds.get(fullName, number).then(function(payload){
			$scope.build = payload.data;
			$scope.task = payload.data.tasks[step-1];

			if (['pending', 'killed'].indexOf($scope.task.state) !== -1) {
				// do nothing
			} else if ($scope.task.state === 'running') {
				// stream the build
				stream();
			} else {

				// fetch the logs for the finished build.
				logs.get(fullName, number, step).then(function(payload){
					var convert = new Filter({stream:false,newline:false});
					var term = document.getElementById("term")
					term.innerHTML = convert.toHtml(payload.data);
				}).catch(function(err){
					$scope.error = err;
				});
			}
		}).catch(function(err){
			$scope.error = err;
		});

		$scope.restart = function() {
			builds.restart(fullName, number).then(function(payload){
				$scope.build = payload.data;
				$scope.task = payload.data.tasks[step-1];
			}).catch(function(err){
				$scope.error = err;
			});
		};

		$scope.cancel = function() {
			builds.cancel(fullName, number).then(function(payload){
				$scope.build = payload.data;
				$scope.task = payload.data.tasks[step-1];
			}).catch(function(err) {
				$scope.error = err;
			});
		};

		$scope.tail = function() {
			tail = !tail;
		};

		feed.subscribe(function(event) {
			if (event.repo.full_name !== fullName) {
				return; // ignore
			}
			if (event.build.number !== parseInt(number)) {
				return; // ignore
			}
			// update the build status
			$scope.build.state = event.build.state;
			$scope.build.started_at = event.build.started_at;
			$scope.build.finished_at = event.build.finished_at;
			$scope.build.duration = event.build.duration;
			$scope.$apply();

			if (!event.task || event.task.number !== step) {
				return; // ignore
			}

			if (event.task.state === 'running' && $scope.task.state !== 'running') {
				stream();
			}

			// update the task status
			$scope.task.state = event.task.state;
			$scope.task.started_at = event.task.started_at;
			$scope.task.finished_at = event.task.finished_at;
			$scope.task.duration = event.task.duration;
			$scope.task.exit_code = event.task.exit_code;
			$scope.$apply();
		});


	}

	angular
		.module('drone')
		.controller('BuildCtrl', BuildCtrl)
		.controller('BuildsCtrl', BuildsCtrl);
})();
