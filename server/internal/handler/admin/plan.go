package admin

import (
	"github.com/gin-gonic/gin"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/resp"
)

// ---- 套餐 ----

func (h *Admin) ListPlans(c *gin.Context) {
	data, err := h.catalog.ListAllPlans(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreatePlan(c *gin.Context) {
	var req model.AdminPlanReq
	if !resp.Bind(c, &req) {
		return
	}
	data, err := h.catalog.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdatePlan(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminPlanReq
	if !resp.Bind(c, &req) {
		return
	}
	if err := h.catalog.UpdatePlan(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeletePlan(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.catalog.DeletePlan(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

// ---- 优惠券 ----

func (h *Admin) ListCoupons(c *gin.Context) {
	data, err := h.catalog.ListAllCoupons(c.Request.Context())
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

func (h *Admin) CreateCoupon(c *gin.Context) {
	var req model.AdminCouponReq
	if !resp.Bind(c, &req) {
		return
	}
	data, err := h.catalog.CreateCoupon(c.Request.Context(), &req)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, data)
}

func (h *Admin) UpdateCoupon(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req model.AdminCouponReq
	if !resp.Bind(c, &req) {
		return
	}
	if err := h.catalog.UpdateCoupon(c.Request.Context(), id, &req); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}

func (h *Admin) DeleteCoupon(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.catalog.DeleteCoupon(c.Request.Context(), id); err != nil {
		resp.Fail(c, err)
		return
	}
	resp.OK(c, nil)
}
