# Singleton: the service-wide NFS configuration.
resource "truenas_nfs_config" "this" {
  servers       = 4
  bindip        = ["0.0.0.0"]
  allow_nonroot = false
  protocols     = ["NFSV3", "NFSV4"]
  v4_krb        = false
  v4_domain     = ""
  mountd_port   = 618
  rpcstatd_port = 871
  rpclockd_port = 32803
}
