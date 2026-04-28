-- Sprint 2 W2.A Vermietung — rollback ACL seed

DELETE FROM permissions WHERE resource IN ('vermietung:object', 'vermietung:rental', 'vermietung:inspection');
