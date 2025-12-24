package httpapi

import (
	"context"
	"net/http"
	"time"

	"fizicamd-backend-go/internal/config"
	"fizicamd-backend-go/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
)

type Server struct {
	DB         *sqlx.DB
	Config     config.Config
	Tokens     services.TokenService
	MetricsHub *services.MetricsHub
}

func NewServer(db *sqlx.DB, cfg config.Config, hub *services.MetricsHub) *Server {
	tokens := services.TokenService{
		Secret:     []byte(cfg.JWTSecret),
		Issuer:     cfg.JWTIssuer,
		AccessTTL:  time.Duration(cfg.AccessTTLSeconds) * time.Second,
		RefreshTTL: time.Duration(cfg.RefreshTTLSeconds) * time.Second,
	}
	return &Server{
		DB:         db,
		Config:     cfg,
		Tokens:     tokens,
		MetricsHub: hub,
	}
}

func (s *Server) Router(ctx context.Context) http.Handler {
	r := chi.NewRouter()
	if len(s.Config.CorsOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.Config.CorsOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
	}

	r.Route("/api", func(api chi.Router) {
		api.Post("/auth/register", s.Register)
		api.Post("/auth/login", s.Login)
		api.Post("/auth/refresh", s.Refresh)
		api.Post("/auth/logout", s.Logout)

		api.Route("/me", func(me chi.Router) {
			me.Use(WithAuth(s.Tokens))
			me.Get("/", s.Me)
			me.Get("/profile", s.Me)
			me.Put("/profile", s.UpdateProfile)
			me.Delete("/", s.DeleteAccount)
			me.Put("/password", s.ChangePassword)
			me.Post("/ping", s.Ping)
		})

		api.Route("/admin", func(admin chi.Router) {
			admin.Use(WithAuth(s.Tokens))
			admin.Use(RequireRole("ADMIN"))
			admin.Get("/metrics/history", s.MetricsHistory)
			admin.Route("/users", func(users chi.Router) {
				users.Get("/", s.ListUsers)
				users.Post("/", s.CreateUser)
				users.Put("/{userId}", s.UpdateUser)
				users.Delete("/{userId}", s.DeleteUser)
				users.Post("/{userId}/roles", s.AssignRole)
				users.Delete("/{userId}/roles/{role}", s.RemoveRole)
			})
			admin.Route("/groups", func(groups chi.Router) {
				groups.Post("/", s.AdminCreateGroup)
				groups.Put("/{groupId}", s.AdminUpdateGroup)
				groups.Delete("/{groupId}", s.AdminDeleteGroup)
				groups.Post("/{groupId}/members", s.AdminAddMember)
				groups.Delete("/{groupId}/members/{userId}", s.AdminRemoveMember)
				groups.Get("/{groupId}", s.AdminGetGroup)
			})
		})

		api.Route("/teacher", func(teacher chi.Router) {
			teacher.Use(WithAuth(s.Tokens))
			teacher.Use(RequireAnyRole("TEACHER", "ADMIN"))

			teacher.Route("/resources", func(resources chi.Router) {
				resources.Get("/", s.TeacherListResources)
				resources.Post("/", s.CreateResource)
				resources.Get("/{resourceId}", s.TeacherResourceDetail)
				resources.Put("/{resourceId}", s.UpdateResource)
				resources.Delete("/{resourceId}", s.DeleteResource)
			})

			teacher.Route("/resource-categories", func(categories chi.Router) {
				categories.Get("/", s.ListCategories)
				categories.Post("/", s.CreateCategory)
				categories.Put("/{code}", s.UpdateCategory)
				categories.Delete("/{code}", s.DeleteCategory)
				categories.Put("/groups/{groupLabel}", s.UpdateGroupLabel)
			})

			teacher.Route("/groups", func(groups chi.Router) {
				groups.Get("/", s.TeacherGroups)
				groups.Get("/{groupId}", s.TeacherGetGroup)
				groups.Put("/{groupId}", s.TeacherUpdateGroup)
				groups.Post("/{groupId}/members", s.TeacherAddMember)
				groups.Delete("/{groupId}/members/{userId}", s.TeacherRemoveMember)
			})
		})

		api.Route("/student/groups", func(groups chi.Router) {
			groups.Use(WithAuth(s.Tokens))
			groups.Use(RequireRole("STUDENT"))
			groups.Get("/", s.StudentGroups)
			groups.Get("/{groupId}", s.StudentGetGroup)
		})

		api.Route("/public", func(pub chi.Router) {
			pub.Get("/search", s.PublicSearch)
			pub.Post("/visits", s.TrackVisit)
			pub.Get("/visits/count", s.VisitCount)

			pub.Route("/resources", func(resources chi.Router) {
				resources.Get("/categories", s.PublicCategories)
				resources.Get("/", s.PublicResources)
				resources.Get("/{slug}", s.PublicResourceDetail)
			})
		})

		api.Route("/media", func(media chi.Router) {
			media.Use(WithAuth(s.Tokens))
			media.Post("/uploads/avatar", s.UploadAvatar)
			media.With(RequireAnyRole("TEACHER", "ADMIN")).Post("/uploads/resource", s.UploadResource)
			media.Get("/assets/{assetId}/content", s.MediaContent)
		})
	})

	r.Get("/ws/metrics", s.MetricsSocket)
	return r
}
