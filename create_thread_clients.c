#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>

void run_client(int client_num) {
	char *client_num_string = malloc(8*sizeof(char) + 10);
	bzero(client_num_string, (8*1+10));

	sprintf(client_num_string, "client_%d", client_num);
	char *args[]={"./executable", "-a", ">", client_num_string, NULL};

	printf("Executing client...");
	int retExec = execvp(args[0], args);

	if (retExec < 0) {
		printf("ERROR: exec failed with status %d and errno %d.\n", retExec, errno);
		exit(errno);
	}

}

int main(int argc, char const* argv[])
{
	if (argc != 1) {
		printf("wrong number of arguments: %d\nexpected 1, since number of clients is an environment variable\n", argc);
		return 1;
	}

	char* p;
	long LONG_NUM_CLIENTS = strtol(getenv("NUM_CLIENTS"), &p, 10);
	if (*p != '\0' || errno != 0) {
		return 1;
	}

	int NUM_CLIENTS = LONG_NUM_CLIENTS;

	printf("Starting %d clients (threads)...", NUM_CLIENTS);

	for(int i = 0; i < NUM_CLIENTS; i++) {
		printf("Creating client %d\n", i);

		if (fork() == 0) {
			run_client(i);
		}

		printf("Created client %d\n", i);
		
		sleep(1);
	}

	for(int i = 0; i < NUM_CLIENTS; i++) {
		wait(NULL);
	}

	return 0;
}
