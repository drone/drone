'use strict';

(function () {

	/**
	 * The LogService provides access to build
	 * log data using REST API calls.
	 */
	function LogService($http, $window) {

		/**
		 * Gets a task logs.
		 *
		 * @param {string} Name of the repository.
		 * @param {number} Number of the build.
		 * @param {number} Number of the task.
		 */
		this.get = function(repoName, number, step) {
			return $http.get('/api/logs/'+repoName+'/'+number+'/'+step);
		};
	}

	angular
		.module('drone')
		.service('logs', LogService);
})();