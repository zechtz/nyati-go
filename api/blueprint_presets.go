package api

import (
	"github.com/google/uuid"
	"github.com/zechtz/nyatictl/config"
)

// GetDefaultBlueprintPreset returns a blueprint preset for a specific application type
func GetDefaultBlueprintPreset(blueprintType string) *Blueprint {
	switch blueprintType {
	case "nodejs":
		return getNodeJSBlueprint()
	case "php":
		return getPHPBlueprint()
	case "python":
		return getPythonBlueprint()
	case "static":
		return getStaticBlueprint()
	default:
		return getBasicBlueprint()
	}
}

// getBasicBlueprint returns a minimal blueprint with simple tasks
func getBasicBlueprint() *Blueprint {
	tasks := []config.Task{
		{
			Name:    "create_release_dir",
			Cmd:     "mkdir -p /var/www/${appname}/releases/${release_version}",
			Expect:  0,
			Message: "Created release directory",
		},
		{
			Name:      "publish",
			Cmd:       "ln -sfn /var/www/${appname}/releases/${release_version} /var/www/${appname}/current",
			Expect:    0,
			Message:   "Deployed successfully to ${env} environment",
			DependsOn: []string{"create_release_dir"},
		},
	}

	return &Blueprint{
		Name:        "Basic Deployment",
		Description: "A basic deployment blueprint with minimal tasks",
		Type:        "custom",
		Version:     "1.0.0",
		Tasks:       assignTaskIDs(tasks),
		Parameters: map[string]string{
			"env": "production",
		},
		IsPublic: true,
	}
}

// getNodeJSBlueprint returns a blueprint for Node.js applications
func getNodeJSBlueprint() *Blueprint {
	tasks := []config.Task{
		{
			Name:    "create_release_dir",
			Cmd:     "mkdir -p /var/www/${appname}/releases/${release_version}",
			Expect:  0,
			Message: "Created release directory",
		},
		{
			Name:      "clone_repository",
			Cmd:       "git clone -b ${branch} ${repository_url} /var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Cloned repository",
			DependsOn: []string{"create_release_dir"},
		},
		{
			Name:      "install_dependencies",
			Cmd:       "${package_manager} install --production",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Installed dependencies",
			DependsOn: []string{"clone_repository"},
		},
		{
			Name:      "build_application",
			Cmd:       "${package_manager} run build",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Built application",
			DependsOn: []string{"install_dependencies"},
		},
		{
			Name:      "setup_env",
			Cmd:       "cp /var/www/${appname}/shared/.env /var/www/${appname}/releases/${release_version}/.env",
			Expect:    0,
			Message:   "Copied environment configuration",
			DependsOn: []string{"clone_repository"},
		},
		{
			Name:      "publish",
			Cmd:       "ln -sfn /var/www/${appname}/releases/${release_version} /var/www/${appname}/current",
			Expect:    0,
			Message:   "Deployed Node.js application successfully",
			DependsOn: []string{"build_application", "setup_env"},
		},
		{
			Name:      "restart_service",
			Cmd:       "systemctl restart ${service_name}",
			Expect:    0,
			AskPass:   true,
			Message:   "Restarted service",
			DependsOn: []string{"publish"},
		},
	}

	return &Blueprint{
		Name:        "Node.js Application",
		Description: "Deployment blueprint for Node.js applications with npm/yarn",
		Type:        "nodejs",
		Version:     "1.0.0",
		Tasks:       assignTaskIDs(tasks),
		Parameters: map[string]string{
			"repository_url":  "git@github.com:username/repo.git",
			"branch":          "main",
			"package_manager": "yarn",
			"service_name":    "${appname}",
			"env":             "production",
		},
		IsPublic: true,
	}
}

// getPHPBlueprint returns a blueprint for PHP applications
func getPHPBlueprint() *Blueprint {
	tasks := []config.Task{
		{
			Name:    "create_release_dir",
			Cmd:     "mkdir -p /var/www/${appname}/releases/${release_version}",
			Expect:  0,
			Message: "Created release directory",
		},
		{
			Name:      "clone_repository",
			Cmd:       "git clone -b ${branch} ${repository_url} /var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Cloned repository",
			DependsOn: []string{"create_release_dir"},
		},
		{
			Name:      "install_dependencies",
			Cmd:       "composer install --no-dev --optimize-autoloader",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Installed dependencies",
			DependsOn: []string{"clone_repository"},
		},
		{
			Name:      "setup_env",
			Cmd:       "cp /var/www/${appname}/shared/.env /var/www/${appname}/releases/${release_version}/.env",
			Expect:    0,
			Message:   "Copied environment configuration",
			DependsOn: []string{"clone_repository"},
		},
		{
			Name:      "run_migrations",
			Cmd:       "php artisan migrate --force",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Ran database migrations",
			DependsOn: []string{"install_dependencies", "setup_env"},
		},
		{
			Name:      "cache_config",
			Cmd:       "php artisan config:cache && php artisan route:cache && php artisan view:cache",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Cached configuration",
			DependsOn: []string{"run_migrations"},
		},
		{
			Name:      "set_permissions",
			Cmd:       "chmod -R 775 storage bootstrap/cache",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Set directory permissions",
			DependsOn: []string{"cache_config"},
		},
		{
			Name:      "publish",
			Cmd:       "ln -sfn /var/www/${appname}/releases/${release_version} /var/www/${appname}/current",
			Expect:    0,
			Message:   "Deployed PHP application successfully",
			DependsOn: []string{"set_permissions"},
		},
		{
			Name:      "restart_php_fpm",
			Cmd:       "sudo service php${php_version}-fpm restart",
			Expect:    0,
			AskPass:   true,
			Message:   "Restarted PHP-FPM",
			DependsOn: []string{"publish"},
		},
	}

	return &Blueprint{
		Name:        "PHP Application",
		Description: "Deployment blueprint for PHP applications with Composer",
		Type:        "php",
		Version:     "1.0.0",
		Tasks:       assignTaskIDs(tasks),
		Parameters: map[string]string{
			"repository_url": "git@github.com:username/repo.git",
			"branch":         "main",
			"php_version":    "8.1",
			"env":            "production",
		},
		IsPublic: true,
	}
}

// getPythonBlueprint returns a blueprint for Python applications
func getPythonBlueprint() *Blueprint {
	tasks := []config.Task{
		{
			Name:    "create_release_dir",
			Cmd:     "mkdir -p /var/www/${appname}/releases/${release_version}",
			Expect:  0,
			Message: "Created release directory",
		},
		{
			Name:      "clone_repository",
			Cmd:       "git clone -b ${branch} ${repository_url} /var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Cloned repository",
			DependsOn: []string{"create_release_dir"},
		},
		{
			Name:      "create_virtualenv",
			Cmd:       "python3 -m venv venv",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Created virtual environment",
			DependsOn: []string{"clone_repository"},
		},
		{
			Name:      "install_dependencies",
			Cmd:       "venv/bin/pip install -r requirements.txt",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Installed dependencies",
			DependsOn: []string{"create_virtualenv"},
		},
		{
			Name:      "setup_env",
			Cmd:       "cp /var/www/${appname}/shared/.env /var/www/${appname}/releases/${release_version}/.env",
			Expect:    0,
			Message:   "Copied environment configuration",
			DependsOn: []string{"clone_repository"},
		},
		{
			Name:      "run_migrations",
			Cmd:       "venv/bin/python manage.py migrate",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Ran database migrations",
			DependsOn: []string{"install_dependencies", "setup_env"},
		},
		{
			Name:      "collect_static",
			Cmd:       "venv/bin/python manage.py collectstatic --noinput",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Collected static files",
			DependsOn: []string{"run_migrations"},
		},
		{
			Name:      "publish",
			Cmd:       "ln -sfn /var/www/${appname}/releases/${release_version} /var/www/${appname}/current",
			Expect:    0,
			Message:   "Deployed Python application successfully",
			DependsOn: []string{"collect_static"},
		},
		{
			Name:      "restart_gunicorn",
			Cmd:       "sudo systemctl restart ${appname}_gunicorn",
			Expect:    0,
			AskPass:   true,
			Message:   "Restarted Gunicorn",
			DependsOn: []string{"publish"},
		},
	}

	return &Blueprint{
		Name:        "Python Application",
		Description: "Deployment blueprint for Python applications with virtualenv",
		Type:        "python",
		Version:     "1.0.0",
		Tasks:       assignTaskIDs(tasks),
		Parameters: map[string]string{
			"repository_url": "git@github.com:username/repo.git",
			"branch":         "main",
			"env":            "production",
		},
		IsPublic: true,
	}
}

// getStaticBlueprint returns a blueprint for static websites
func getStaticBlueprint() *Blueprint {
	tasks := []config.Task{
		{
			Name:    "create_release_dir",
			Cmd:     "mkdir -p /var/www/${appname}/releases/${release_version}",
			Expect:  0,
			Message: "Created release directory",
		},
		{
			Name:      "clone_repository",
			Cmd:       "git clone -b ${branch} ${repository_url} /var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Cloned repository",
			DependsOn: []string{"create_release_dir"},
		},
		{
			Name:      "install_dependencies",
			Cmd:       "npm install",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Installed dependencies",
			DependsOn: []string{"clone_repository"},
		},
		{
			Name:      "build_site",
			Cmd:       "npm run build",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Built static website",
			DependsOn: []string{"install_dependencies"},
		},
		{
			Name:      "set_permissions",
			Cmd:       "chmod -R 755 ${build_dir}",
			Dir:       "/var/www/${appname}/releases/${release_version}",
			Expect:    0,
			Message:   "Set directory permissions",
			DependsOn: []string{"build_site"},
		},
		{
			Name:      "publish",
			Cmd:       "ln -sfn /var/www/${appname}/releases/${release_version}/${build_dir} /var/www/${appname}/current",
			Expect:    0,
			Message:   "Deployed static website successfully",
			DependsOn: []string{"set_permissions"},
		},
	}

	return &Blueprint{
		Name:        "Static Website",
		Description: "Deployment blueprint for static websites",
		Type:        "static",
		Version:     "1.0.0",
		Tasks:       assignTaskIDs(tasks),
		Parameters: map[string]string{
			"repository_url": "git@github.com:username/repo.git",
			"branch":         "main",
			"build_dir":      "dist",
			"env":            "production",
		},
		IsPublic: true,
	}
}

func assignTaskIDs(tasks []config.Task) []config.Task {
	for i := range tasks {
		if tasks[i].ID == "" {
			tasks[i].ID = uuid.NewString()
		}
	}
	return tasks
}
