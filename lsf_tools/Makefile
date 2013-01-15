CC = g++
CFLAGS = -I/usr/local/lsf/7.0/include -std=gnu++0x -DPOSIX_SOURCE -g
OBJ = main.o
LDFLAGS = -L/usr/local/lsf/7.0/lib -Wl,-rpath /usr/local/lsf/7.0/lib -lnsl -llsf -lm -lbat

all: lsf_fairshare

%.o: %.cpp
	$(CC) $(CFLAGS) -c $<

lsf_fairshare: $(OBJ)
	$(CC) -o lsf_fairshare $(LDFLAGS) $(OBJ)

all: lsf_fairshare

