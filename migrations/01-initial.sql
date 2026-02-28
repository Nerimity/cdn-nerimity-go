CREATE TABLE expire_file (
    "fileId" TEXT NOT NULL,
    "groupId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "ExpireFile_pkey" PRIMARY KEY ("fileId")
);
