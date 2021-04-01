// ARGS PASSED THROUGH ENVIRONMENT VARIABLES:
// NUM_CLIENTS -> number of clients to launch
// REGION -> client region
// CLIENTS_TIMEOUT -> duration of client interaction

#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>
#include <sys/wait.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>

void run_client(int client_num, char *clients_region, char *timeout_string,
				char *logs_dir, char *executable_path)
{
	struct timespec tm;
	clock_gettime(CLOCK_MONOTONIC, &tm);
	// xor-ing with tv_nsec >> 31 to ensure even low precision clocks vary
	// the low 32 bits
	srand((unsigned)(tm.tv_sec ^ tm.tv_nsec ^ (tm.tv_nsec >> 31)));

	int randomTime = rand() % 10 + 1;
	sleep(randomTime);

	char *client_num_string = malloc(11 * sizeof(char));
	sprintf(client_num_string, "%d", client_num);
	int string_full_size = strlen(client_num_string) + 8 + strlen(logs_dir) + 4 + 1;
	char *client_filename = malloc(string_full_size * sizeof(char));

	sprintf(client_filename, "%s/client_%d.log", logs_dir, client_num);

	char *args[] = {executable_path, "-a", "-n", client_num_string,
					"-r", clients_region, "-t", timeout_string, "-ld", logs_dir,
					NULL};

	int out;
	out = open(client_filename, O_WRONLY | O_CREAT | O_APPEND, 0666);

	dup2(out, fileno(stdout));
	dup2(out, fileno(stderr));

	close(out);

	printf("Executing client...\n");
	fflush(stdout);
	int retExec = execvp(args[0], args);

	if (retExec < 0)
	{
		printf("ERROR: exec failed with status %d and errno %d.\n", retExec,
			   errno);
		fflush(stdout);
		exit(errno);
	}

	free(client_num_string);
	free(client_filename);
}

void *wait_for_client(void *args)
{
	int *args_int = (int *)args;

	int client_num = args_int[0];
	pid_t fork_pid = args_int[1];

	int status;
	waitpid(fork_pid, &status, 0);

	printf("Client %d crashed! Check logs.\n", client_num);
	fflush(stdout);

	return NULL;
}

int main(int argc, char const *argv[])
{
	if (argc != 1)
	{
		printf("wrong number of arguments: %d\nall arguments should be passed as environment variables\n", argc - 1);
		fflush(stdout);
		return 1;
	}

	char *p;
	long LONG_NUM_CLIENTS = strtol(getenv("NUM_CLIENTS"), &p, 10);
	if (*p != '\0' || errno != 0)
	{
		return 1;
	}

	char *clients_region = getenv("REGION");

	int NUM_CLIENTS = LONG_NUM_CLIENTS;
	pthread_t waiting_threads_ids[NUM_CLIENTS];
	int thread_args[NUM_CLIENTS][2];

	char *timeout_string = getenv("CLIENTS_TIMEOUT");

	char *logs_dir = getenv("LOGS_DIR");
	if (logs_dir == NULL)
	{
		logs_dir = "/logs";
	}

	printf("Starting %d clients (threads)...\n", NUM_CLIENTS);
	fflush(stdout);

	char *project_dir = getenv("NOVAPOKEMON");
	char *executable_string = "client/executable";
	char *executable_path =
		malloc((strlen(project_dir) + strlen(executable_string) + 1) *
			   sizeof(char));
	sprintf(executable_path, "%s/%s", project_dir, executable_string);

	for (int i = 0; i < NUM_CLIENTS; i++)
	{
		printf("Creating client %d\n", i);
		fflush(stdout);

		pid_t fork_id = fork();

		if (fork_id == 0)
		{
			// CHILD
			run_client(i, clients_region, timeout_string, logs_dir,
					   executable_path);
		}

		thread_args[i][0] = i;
		thread_args[i][1] = fork_id;

		pthread_create(&waiting_threads_ids[i], NULL, wait_for_client,
					   thread_args[i]);

		printf("Created client %d\n", i);
		fflush(stdout);
	}

	printf("Finished creating clients.\n");

	for (int i = 0; i < NUM_CLIENTS; i++)
	{
		pthread_join(waiting_threads_ids[i], NULL);
		printf("Finished thread %d", i);
	}

	printf("Thread crashed... Check logs.\n");
	fflush(stdout);

	return 0;
}
