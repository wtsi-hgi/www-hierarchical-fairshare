#include <cstdint>
#include <lsf/lsbatch.h>
#include <cstdlib>
#include <map>
#include <cstring>
#include <string>

class ShareAcct
{
public:
	void print()
	{
		printf( "{Group:'%s',Parent:'%s',Percentage:%d,Priority:%f}\n", group.c_str(), parent.c_str(), shares, priority);
	}
private:
	std::string group;
	std::string parent;
	int shares;
	float priority;

	static std::pair<std::string,std::string>* digestPath( const char *input )
	{
		char* manip = strdup(input);
		ssize_t orig_size = strlen(manip);
		int tokens_found = 0;
		for (ssize_t loop_iter = 0; loop_iter < strlen(manip); loop_iter++)
		{
			if (manip[loop_iter] == '/') {
				tokens_found++;
			}
		}
		char** tokens = (char**) malloc(sizeof(char*)*(tokens_found+1));
		tokens[0] = manip;
		int token_pos = 1;
		for (ssize_t loop_iter = 0; loop_iter < orig_size; loop_iter++)
		{
			if (manip[loop_iter] == '/') {
				manip[loop_iter] = '\0';
				tokens[token_pos] = manip+loop_iter+1;
				token_pos++;
			}
		}
		return new std::pair<std::string,std::string>(tokens[tokens_found], tokens[tokens_found-1]);
	}
public:
	ShareAcct(const struct shareAcctInfoEnt *curr_share)
	{
		std::pair<std::string, std::string>* path = digestPath(curr_share->shareAcctPath);
		group = path->first;
		parent = path->second;
		delete path;

		shares = curr_share->shares;
		priority = curr_share->priority;
	}

	const char* getParent() { return parent.c_str(); }
};

void printShareAcct( const struct shareAcctInfoEnt *curr_share )
{
	char* manip = strdup(curr_share->shareAcctPath);
	ssize_t orig_size = strlen(manip);
	int tokens_found = 0;
	for (ssize_t loop_iter = 0; loop_iter < strlen(manip); loop_iter++)
	{
		if (manip[loop_iter] == '/') {
			tokens_found++;
		}
	}
	char** tokens = (char**) malloc(sizeof(char*)*(tokens_found+1));
	tokens[0] = manip;
	int token_pos = 1;
        for (ssize_t loop_iter = 0; loop_iter < orig_size; loop_iter++)
        {
                if (manip[loop_iter] == '/') {
			manip[loop_iter] = '\0';
                        tokens[token_pos] = manip+loop_iter+1;
			token_pos++;
                }
        }

	printf( "{Group:'%s',Parent:'%s',Percentage:%d,Priority:%f}\n", tokens[tokens_found], tokens[tokens_found-1], curr_share->shares, curr_share->priority);
}

int main(int argc, char** argv)
{
	// Initialise our connection to LSF
	if (lsb_init(argv[0]) < 0) {
		lsb_perror("lsb_init() failed");
		return -1;
	}

	// If queue argument isn't defined then just show data from normal job queue
	char* queues;
	if (argc < 2) {
		queues = strdup("normal");
	}
	else {
		queues = argv[1];
	}

	// Query LSF
	int num_queues = 1;
	struct queueInfoEnt *info = lsb_queueinfo(&queues, &num_queues, NULL, NULL, 0);
	// For each queue in output (should only be one) process data
	for (int i = 0; i < num_queues; i++) {
		struct queueInfoEnt *curr_info = &info[i];
		std::multimap<std::string,ShareAcct*> *parent_map = new std::multimap<std::string,ShareAcct*>();
		// SKUNKWORKS SOLUTION
		for (int j = 1; j < curr_info->numOfSAccts; j++) {
			struct shareAcctInfoEnt *curr_share = &curr_info->shareAccts[j];
			ShareAcct* acct = new ShareAcct( curr_share );
			parent_map->insert(std::make_pair<std::string,ShareAcct*>(std::string(acct->getParent()), acct));
		}
		// /SKUNKWORKS SOLUTION

		// If there are actually any share accounts defined for this queue write result
		// It's done this way because JSON doesn't support commas at the end of the last list item
		if (curr_info->numOfSAccts > 0) {
			printf("[\n");
			// Write first element
			struct shareAcctInfoEnt *curr_share = &curr_info->shareAccts[0];
			printShareAcct( curr_share );
			// Write subsequent elements
			for (int j = 1; j < curr_info->numOfSAccts; j++) {
				printf(",\n");
				struct shareAcctInfoEnt *curr_share = &curr_info->shareAccts[j];
				printShareAcct( curr_share );
			}
			printf("]\n");
		}
	}
	return 0;
}

