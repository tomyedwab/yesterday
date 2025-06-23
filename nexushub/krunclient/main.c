#include <libkrun.h>
#include <stdio.h>
#include <string.h>

extern char **environ;

int main(int argc, char *argv[])
{
	if (argc != 3) {
		fprintf(stderr, "Usage: %s <root_path> <local_port>\n", argv[0]);
		return 1;
	}

	// Build port mapping string dynamically
	char port_mapping[32];
	snprintf(port_mapping, sizeof(port_mapping), "%s:80", argv[2]);
	const char *port_map[] = {port_mapping, NULL};

	char * envp[] = {
		"HOST=",
		"INTERNAL_SECRET=",
		0,
	};
	for (int i = 0; environ[i] != NULL; ++i) {
	    if (!strncmp(environ[i], "HOST=", 5)) {
			printf("Setting HOST environment variable to %s\n", &environ[i][5]);
	        envp[0] = strdup(environ[i]);
	    }
	    if (!strncmp(environ[i], "INTERNAL_SECRET=", 16)) {
			printf("Setting INTERNAL_SECRET environment variable\n");
	        envp[1] = strdup(environ[i]);
	    }
	}

	int ctx_id = krun_create_ctx();
	printf("Initializing VM context...\n");
	krun_set_vm_config(ctx_id, 1, 512);
	printf("Setting VM root to %s\n", argv[1]);
	krun_set_root(ctx_id, argv[1]);
	printf("Mapping TCP ports %s\n", port_mapping);
	krun_set_port_map(ctx_id, port_map);
	printf("Executing /bin/app in VM...\n");
	krun_set_exec(ctx_id, "/bin/app", 0, (const char* const*)&envp[0]);
	krun_start_enter(ctx_id);

	return 0;
}
