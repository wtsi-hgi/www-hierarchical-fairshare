#include <cstdint>
#include <lsf/lsbatch.h>
#include <cstdlib>
#include <list>
#include <map>
#include <cstring>
#include <string>

class ShareAcct
{
  public:
    static void print_structure() {
      printf("['Group', 'Parent', 'Percentage', 'Priority']");
    }
    void print(bool is_parent) {
      if (is_parent) {
        printf( "['%s','%s',%f,%f]\n", group.c_str(), parent.c_str(), proportion, priority);
      } else {
        printf( "['%s@%s','%s',%f,%f]\n", group.c_str(), parent.c_str(), parent.c_str(), proportion, priority);
      }
    }
  private:
    std::string group;
    std::string parent;
    int shares;
    float proportion;
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
      proportion = 1.123;
    }

    const char* getParent() { return parent.c_str(); }
    const char* getGroup() { return group.c_str(); }
    const float getPriority() { return priority; }
    const float getShares() { return shares; }
    const float getProportion() { return proportion; }

    void setProportion(const float p) { proportion = p; }
};

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
    std::map<std::string, float> parent_map;
    std::list<ShareAcct*> accounts;
    // SKUNKWORKS SOLUTION
    for (int j = 1; j < curr_info->numOfSAccts; j++) {
      struct shareAcctInfoEnt *curr_share = &curr_info->shareAccts[j];
      ShareAcct* acct = new ShareAcct( curr_share );
      std::string parent = std::string(acct->getParent());
      // If the parent is already in the map, sum the priorities. Otherwise, add it in with the priority of the first child.
      if (parent_map.count(parent) > 0) {
        parent_map[parent] += acct->getShares();
      } else {
        parent_map[parent] = acct->getShares();
      }

      accounts.push_back(acct);
    }

    // Now walk back through the accounts (reverse order) and first normalise shares,
    // then multiply that by the value we calculated for the parents. We do this recursively until
    // we hit the bottom.
    std::map<std::string, float> share_map;
    for (auto it = accounts.begin(); it != accounts.end(); it++) {
      ShareAcct* a = *it;
      if (parent_map.count(a->getParent()) > 0) {
        if (share_map.count(a->getParent()) > 0) {
          share_map[a->getGroup()] = ((a->getShares() / parent_map[a->getParent()]) * share_map[a->getParent()]);
        } else {
          share_map[a->getGroup()] = (a->getShares() / parent_map[a->getParent()]);
        }
      } else {
        share_map[a->getGroup()] = 1;
      }

      a->setProportion(share_map[a->getGroup()]);

    }


    // /SKUNKWORKS SOLUTION
    std::string root = accounts.front()->getParent();

    // If there are actually any share accounts defined for this queue write result
    // It's done this way because JSON doesn't support commas at the end of the last list item
    if (curr_info->numOfSAccts > 0) {
      printf("[\n");
      ShareAcct::print_structure();
      // Write the root node
      printf("\n,\n['%s','' , 1, 1]", root.c_str());
      // Write elements
      for (auto it = accounts.begin(); it != accounts.end(); it++) {
        printf(",\n");
        (*it)->print(parent_map.count((*it)->getGroup()) > 0);
      }
      printf("]\n");
    }
  }
  return 0;
}
