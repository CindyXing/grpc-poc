func checkService(ctx context.Context, serviceName string) (healthpb.HealthCheckResponse_ServingStatus, error) {

	glog.V(10).Infof("establishing connection")
	connStart := time.Now()
	dialCtx, dialCancel := context.WithTimeout(ctx, cfg.flConnTimeout)
	defer dialCancel()
	conn, err := grpc.DialContext(dialCtx, cfg.flGrpcServerAddr, opts...)
	if err != nil {
		if err == context.DeadlineExceeded {
			glog.Warningf("timeout: failed to connect service %s within %s", cfg.flGrpcServerAddr, cfg.flConnTimeout)
		} else {
			glog.Warningf("error: failed to connect service at %s: %+v", cfg.flGrpcServerAddr, err)
		}
		return healthpb.HealthCheckResponse_UNKNOWN, NewGrpcProbeError(StatusConnectionFailure, "StatusConnectionFailure")
	}
	connDuration := time.Since(connStart)
	defer conn.Close()
	glog.V(10).Infof("connection established %v", connDuration)

	rpcStart := time.Now()
	rpcCtx, rpcCancel := context.WithTimeout(ctx, cfg.flRPCTimeout)
	defer rpcCancel()

	glog.V(10).Infoln("Running HealthCheck for service: ", serviceName)

	resp, err := healthpb.NewHealthClient(conn).Check(rpcCtx, &healthpb.HealthCheckRequest{Service: serviceName})
	if err != nil {
		// first handle and return gRPC-level errors
		if stat, ok := status.FromError(err); ok && stat.Code() == codes.Unimplemented {
			glog.Warningf("error: this server does not implement the grpc health protocol (grpc.health.v1.Health)")
			return healthpb.HealthCheckResponse_UNKNOWN, NewGrpcProbeError(StatusUnimplemented, "StatusUnimplemented")
		} else if stat, ok := status.FromError(err); ok && stat.Code() == codes.DeadlineExceeded {
			glog.Warningf("error timeout: health rpc did not complete within ", cfg.flRPCTimeout)
			return healthpb.HealthCheckResponse_UNKNOWN, NewGrpcProbeError(StatusRPCFailure, "StatusRPCFailure")
		} else if stat, ok := status.FromError(err); ok && stat.Code() == codes.NotFound {
			// wrap a grpC NOT_FOUND as grpcProbeError.
			// https://github.com/grpc/grpc/blob/master/doc/health-checking.md
			// if the service name is not registerered, the server returns a NOT_FOUND GPRPC status. 
			// the Check for a not found should "return nil, status.Error(codes.NotFound, "unknown service")"
			glog.Warningf("error Service Not Found %v", err )
			return healthpb.HealthCheckResponse_SERVICE_UNKNOWN, NewGrpcProbeError(StatusServiceNotFound, "StatusServiceNotFound")
		} else {
			glog.Warningf("error: health rpc failed: ", err)
		}		
	}
	rpcDuration := time.Since(rpcStart)
	// otherwise, retrurn gRPC-HC status
	glog.V(10).Infof("time elapsed: connect=%s rpc=%s", connDuration, rpcDuration)

	return resp.GetStatus(), nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {

	var serviceName string
	if (cfg.flServiceName != "") {
		serviceName = cfg.flServiceName
	}	
	keys, ok := r.URL.Query()["serviceName"]
	if ok && len(keys[0]) > 0 {
		serviceName = keys[0]
	}

	resp, err := checkService(r.Context(), serviceName)
	// first handle errors derived from gRPC-codes
	if err != nil {
        if pe, ok := err.(*GrpcProbeError); ok {
			glog.Errorf("HealtCheck Probe Error: %v", pe.Error())
			switch pe.Code {
			case StatusConnectionFailure:
				http.Error(w, err.Error(), http.StatusBadGateway)
			case StatusRPCFailure:
				http.Error(w, err.Error(), http.StatusBadGateway)
			case StatusUnimplemented:
				http.Error(w, err.Error(), http.StatusNotImplemented)				
			case StatusServiceNotFound:
				http.Error(w, fmt.Sprintf("%s ServiceNotFound", cfg.flServiceName), http.StatusNotFound)									
			default:
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			return
		}
	}

	// then grpc-hc codes
	glog.Infof("%s %v",  cfg.flServiceName,  resp.String())
	switch resp {
	case healthpb.HealthCheckResponse_SERVING:
		fmt.Fprintf(w, "%s %v", cfg.flServiceName, resp)
	case healthpb.HealthCheckResponse_NOT_SERVING:
		http.Error(w, fmt.Sprintf("%s %v", cfg.flServiceName, resp.String()), http.StatusBadGateway)	
	case healthpb.HealthCheckResponse_UNKNOWN:
		http.Error(w, fmt.Sprintf("%s %v", cfg.flServiceName, resp.String()), http.StatusBadGateway)	
	case healthpb.HealthCheckResponse_SERVICE_UNKNOWN:
		http.Error(w, fmt.Sprintf("%s %v", cfg.flServiceName, resp.String()), http.StatusNotFound)	
	}
}
